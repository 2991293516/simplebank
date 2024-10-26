package api

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httptest"
	"reflect"
	mockdb "simplebank/db/mock"
	db "simplebank/db/sqlc"
	"simplebank/util"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"
)

type UserEqMatcher struct {
	arg      db.CreateUserParams
	password string
}

// 这个x是 经过CreateApi执行数据库操作前的逻辑处理过后的请求体
func (e UserEqMatcher) Matches(x interface{}) bool {
	// 对arg进行断言
	arg, ok := x.(db.CreateUserParams)
	if !ok {
		return false
	}

	// 密码进行校验
	flag := util.CheckPassword(e.password, arg.HashedPassword)
	if !flag {
		return false
	}

	e.arg.HashedPassword = arg.HashedPassword

	return reflect.DeepEqual(e.arg, arg)
}

func (e UserEqMatcher) String() string {
	return fmt.Sprintf("matches arg %v and password %v", e.arg, e.password)
}

func UserEq(arg db.CreateUserParams, password string) gomock.Matcher {
	return UserEqMatcher{arg, password}
}

func TestCreateUser(t *testing.T) {
	user, password := randomUser(t)

	testCases := []struct {
		name          string
		body          gin.H
		buildStubs    func(store *mockdb.MockStore)
		checkResponse func(*httptest.ResponseRecorder)
	}{
		{
			name: "OK",
			body: gin.H{
				"username":        user.Username,
				"hashed_password": password,
				"full_name":       user.FullName,
				"email":           user.Email,
			},
			buildStubs: func(store *mockdb.MockStore) {
				createUserParams := db.CreateUserParams{
					Username: user.Username,
					FullName: user.FullName,
					Email:    user.Email,
				}
				store.EXPECT().
					CreateUser(gomock.Any(), UserEq(createUserParams, password)).
					Times(1).
					Return(user, nil)
			},
			checkResponse: func(recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusOK, recorder.Code)
				requireBodyMatchUser(t, recorder.Body, user)
			},
		},
		{
			name: "BadRequest",
			body: gin.H{
				"username":        user.Username,
				"hashed_password": password,
			},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().
					CreateUser(gomock.Any(), gomock.Any()).
					Times(0)
			},
			checkResponse: func(recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusBadRequest, recorder.Code)
			},
		},
		{
			name: "InternalServerError",
			body: gin.H{
				"username":        user.Username,
				"hashed_password": password,
				"full_name":       user.FullName,
				"email":           user.Email,
			},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().
					CreateUser(gomock.Any(), gomock.Any()).
					Times(1).
					Return(db.User{}, sql.ErrConnDone)
			},
			checkResponse: func(recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusInternalServerError, recorder.Code)
			},
		},
	}

	for i := range testCases {
		tc := testCases[i]

		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)

			store := mockdb.NewMockStore(ctrl)
			tc.buildStubs(store)

			server := newTestServer(t, store)
			recorder := httptest.NewRecorder()

			//解析数据
			data, err := json.Marshal(tc.body)
			require.NoError(t, err)

			url := "/users"
			request, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(data))
			require.NoError(t, err)

			server.router.ServeHTTP(recorder, request)
			tc.checkResponse(recorder)
		})
	}
}

func TestLoginUser(t *testing.T) {
	user, password := randomUser(t)

	testCases := []struct {
		name          string
		body          gin.H
		buildStubs    func(store *mockdb.MockStore)
		checkResponse func(server *Server, recoder *httptest.ResponseRecorder)
	}{
		{
			name: "OK",
			body: gin.H{
				"username": user.Username,
				"password": password,
			},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().
					GetUser(gomock.Any(), gomock.Eq(user.Username)).
					Times(1).
					Return(user, nil)
			},
			checkResponse: func(server *Server, recoder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusOK, recoder.Code)

				bodyBytes, err := ioutil.ReadAll(recoder.Body)
				require.NoError(t, err)

				var response loginUserRespose
				err = json.Unmarshal(bodyBytes, &response)
				require.NoError(t, err)
				require.NotEmpty(t, response)

				require.Equal(t, user.Username, response.User.Username)
				require.Equal(t, user.FullName, response.User.FullName)
				require.Equal(t, user.Email, response.User.Email)

				//检验token是否有效
				require.NotEmpty(t, response.Token)
				_, err = server.tokenMaker.VerifyToken(response.Token)
				require.NoError(t, err)
			},
		},
		{
			name: "ShortSecret",
			body: gin.H{
				"username": user.Username,
				"password": "123",
			},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().
					GetUser(gomock.Any(), gomock.Any()).
					Times(0)
			},
			checkResponse: func(server *Server, recoder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusBadRequest, recoder.Code)
			},
		},
		{
			name: "NotFound User",
			body: gin.H{
				"username": user.Username,
				"password": password,
			},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().
					GetUser(gomock.Any(), gomock.Eq(user.Username)).
					Times(1).
					Return(db.User{}, sql.ErrNoRows)
			},
			checkResponse: func(server *Server, recoder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusNotFound, recoder.Code)
			},
		},
		{
			name: "InternalServerError",
			body: gin.H{
				"username": user.Username,
				"password": password,
			},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().
					GetUser(gomock.Any(), gomock.Eq(user.Username)).
					Times(1).
					Return(db.User{}, sql.ErrConnDone)
			},
			checkResponse: func(server *Server, recoder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusInternalServerError, recoder.Code)
			},
		},
		{
			name: "InvalidPassword",
			body: gin.H{
				"username": user.Username,
				"password": "wrong_password",
			},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().
					GetUser(gomock.Any(), gomock.Eq(user.Username)).
					Times(1).
					Return(user, nil)
			},
			checkResponse: func(server *Server, recoder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusUnauthorized, recoder.Code)
			},
		},
	}

	for i := range testCases {
		tc := testCases[i]

		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			store := mockdb.NewMockStore(ctrl)
			tc.buildStubs(store)

			server := newTestServer(t, store)
			recoder := httptest.NewRecorder()

			data, err := json.Marshal(tc.body)
			require.NoError(t, err)

			url := "/users/login"
			request, err := http.NewRequest(http.MethodGet, url, bytes.NewBuffer(data))
			require.NoError(t, err)

			server.router.ServeHTTP(recoder, request)
			tc.checkResponse(server, recoder)

			log.Println("response:")
		})
	}
}

func randomUser(t *testing.T) (user db.User, password string) {
	password = util.RandomString(6)
	hashPassword, err := util.HashedPassword(password)
	require.NoError(t, err)

	user = db.User{
		Username:       util.RandomString(6),
		HashedPassword: hashPassword,
		FullName:       util.RandomString(6),
		Email:          util.RandomEmail(),
	}

	return
}

func requireBodyMatchUser(t *testing.T, body *bytes.Buffer, user db.User) {
	data, err := io.ReadAll(body)
	require.NoError(t, err)

	var gotUser db.User
	err = json.Unmarshal(data, &gotUser)
	require.NoError(t, err)

	require.Equal(t, user.Username, gotUser.Username)
	require.Equal(t, user.FullName, gotUser.FullName)
	require.Equal(t, user.Email, gotUser.Email)
}
