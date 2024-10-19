package db

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

// 测试转账功能
func TestTransferTx(t *testing.T) {
	store := NewStore(testDB)

	account1 := createRandomAccount(t)
	account2 := createRandomAccount(t)

	//跑五个线程并发转账
	n := 5
	amount := int64(10)

	//使用通道来控制接受各个线程的结果
	errs := make(chan error)
	results := make(chan TransferTxResult)

	for i := 0; i < n; i++ {
		go func() {
			//用户1向用户2转10块
			result, err := store.TransferTx(context.Background(), TransferTxParams{
				FromAccountID: account1.ID,
				ToAccountID:   account2.ID,
				Amount:        amount,
			})

			//转账完成后将结果和错误信息发送到通道
			errs <- err
			results <- result
		}()
	}

	existed := make(map[int]bool)

	//循环n此，利用chan堵塞的特点等待所有线程执行完成，并对结果进行校验
	for i := 0; i < n; i++ {
		err := <-errs
		require.NoError(t, err)

		result := <-results
		require.NotEmpty(t, result)

		//校验转账记录
		transfer := result.Transfer
		require.NotEmpty(t, transfer)
		require.Equal(t, transfer.FromAccountID, account1.ID)
		require.Equal(t, transfer.ToAccountID, account2.ID)
		require.Equal(t, transfer.Amount, amount)
		require.NotZero(t, transfer.ID)
		require.NotZero(t, transfer.CreatedAt)

		//判断数据库是否存在此记录
		_, err = store.GetTransfer(context.Background(), transfer.ID)
		require.NoError(t, err)

		//校验用户1的账目记录
		fromEntry := result.FromEntry
		require.NotEmpty(t, fromEntry)
		require.NotZero(t, fromEntry.ID)
		require.NotZero(t, fromEntry.CreatedAt)
		require.Equal(t, fromEntry.Amount, -amount)
		require.Equal(t, fromEntry.AccountID, account1.ID)

		//判断数据库是否存在此记录
		_, err = store.GetEntry(context.Background(), fromEntry.ID)
		require.NoError(t, err)

		//校验用户2的账目记录
		toEntry := result.ToEntry
		require.NotEmpty(t, toEntry)
		require.NotZero(t, toEntry.ID)
		require.NotZero(t, toEntry.CreatedAt)
		require.Equal(t, toEntry.Amount, amount)
		require.Equal(t, toEntry.AccountID, account2.ID)

		//判断数据库是否存在此记录
		_, err = store.GetEntry(context.Background(), toEntry.ID)
		require.NoError(t, err)

		//校验账户信息
		fromAccount := result.FromAccount
		require.NotEmpty(t, fromAccount)
		require.Equal(t, fromAccount.ID, account1.ID)

		toAccount := result.ToAccount
		require.NotEmpty(t, toAccount)
		require.Equal(t, toAccount.ID, account2.ID)

		//校验账户余额

		//两个账户的余额变化应该相同
		diff1 := account1.Balance - fromAccount.Balance
		diff2 := toAccount.Balance - account2.Balance
		require.Equal(t, diff1, diff2)

		//校验余额变化是否正确
		require.True(t, diff1 > 0)
		require.True(t, diff1%amount == 0) //转账金额必须为amount的整数倍

		//并且diff1%amount的值不可以重复
		k := int(diff1 / amount)
		require.True(t, k >= 1 && k <= n)
		require.NotContains(t, existed, k)

		existed[k] = true
	}

	//交易完后，校验所有余额是否正确
	updateAccount1, err := store.GetAccount(context.Background(), account1.ID)
	require.NoError(t, err)

	updateAccount2, err := store.GetAccount(context.Background(), account2.ID)
	require.NoError(t, err)

	require.Equal(t, (updateAccount1.Balance + int64(n)*amount), account1.Balance)
	require.Equal(t, (updateAccount2.Balance - int64(n)*amount), account2.Balance)
}

func TestTransferTxDeadLoad(t *testing.T) {
	store := NewStore(testDB)

	account1 := createRandomAccount(t)
	account2 := createRandomAccount(t)

	n := 10
	amount := int64(10)

	//使用通道来控制接受各个线程的结果
	errs := make(chan error)

	for i := 0; i < n; i++ {
		fromAccountID := account1.ID
		toAccountID := account2.ID

		//模拟两个用户之间同时进行转账操作
		if i%2 == 0 {
			fromAccountID = account1.ID
			toAccountID = account2.ID
		} else {
			fromAccountID = account2.ID
			toAccountID = account1.ID
		}

		go func() {
			//用户1向用户2转10块
			_, err := store.TransferTx(context.Background(), TransferTxParams{
				FromAccountID: fromAccountID,
				ToAccountID:   toAccountID,
				Amount:        amount,
			})

			//转账完成后将结果和错误信息发送到通道
			errs <- err
		}()
	}

	//循环n此，利用chan堵塞的特点等待所有线程执行完成，并对结果进行校验
	for i := 0; i < n; i++ {
		err := <-errs
		require.NoError(t, err)
	}

	//交易完后，校验所有余额是否正确
	updateAccount1, err := store.GetAccount(context.Background(), account1.ID)
	require.NoError(t, err)

	updateAccount2, err := store.GetAccount(context.Background(), account2.ID)
	require.NoError(t, err)

	require.Equal(t, updateAccount1.Balance, account1.Balance)
	require.Equal(t, updateAccount2.Balance, account2.Balance)
}
