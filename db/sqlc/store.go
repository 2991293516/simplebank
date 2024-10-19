package db

import (
	"context"
	"database/sql"
	"fmt"
)

type Store struct {
	*Queries
	db *sql.DB
}

func NewStore(db *sql.DB) *Store {
	return &Store{
		db:      db,
		Queries: New(db),
	}
}

// 执行事务，为了安全，此方法不对外暴露（为什么要独立出来？因为涉及到的事物的代码结构基本一致，所以独立出来可以复用代码）
func (store *Store) execTx(ctx context.Context, fn func(*Queries) error) error {
	//开启事物
	tx, err := store.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}

	//执行具体操作
	q := New(tx)
	err = fn(q)
	if err != nil {
		//出错了要回滚事物
		rbErr := tx.Rollback()
		if rbErr != nil {
			return fmt.Errorf("txErr: %v , rbErr: %v", err, rbErr)
		}
		return err
	}

	//提交事物
	return tx.Commit()
}

/**
用户转账事物
*/

// 转让记录DTO
type TransferTxParams struct {
	FromAccountID int64 `json:"from_account_id"`
	ToAccountID   int64 `json:"to_account_id"`
	Amount        int64 `json:"amount"`
}

// 转让记录VO
type TransferTxResult struct {
	Transfer    Transfer `json:"transfer"`
	FromAccount Account  `json:"from_account"`
	ToAccount   Account  `json:"to_account"`
	FromEntry   Entry    `json:"from_entry"`
	ToEntry     Entry    `json:"to_entry"`
}

func (store *Store) TransferTx(ctx context.Context, arg TransferTxParams) (TransferTxResult, error) {
	var result TransferTxResult

	err := store.execTx(ctx, func(q *Queries) error {
		var err error

		// 创建转让记录
		result.Transfer, err = q.CreateTransfer(ctx, CreateTransferParams{
			FromAccountID: arg.FromAccountID,
			ToAccountID:   arg.ToAccountID,
			Amount:        arg.Amount,
		})
		if err != nil {
			return err
		}

		// 为出钱方创建账户条目
		result.FromEntry, err = q.CreateEntry(ctx, CreateEntryParams{
			AccountID: arg.FromAccountID,
			Amount:    -arg.Amount,
		})
		if err != nil {
			return err
		}

		// 为收钱方创建账户条目
		result.ToEntry, err = q.CreateEntry(ctx, CreateEntryParams{
			AccountID: arg.ToAccountID,
			Amount:    arg.Amount,
		})
		if err != nil {
			return err
		}

		// 让id大的用户现更新余额，避免在用户1更用户2同时互相转账时因顺序问题导致死锁
		if arg.FromAccountID > arg.ToAccountID {
			AddMoney(ctx, q, arg.FromAccountID, -arg.Amount, arg.ToAccountID, arg.Amount)
		} else {
			AddMoney(ctx, q, arg.ToAccountID, arg.Amount, arg.FromAccountID, -arg.Amount)
		}

		result.FromAccount, err = q.GetAccount(ctx, arg.FromAccountID)
		if err != nil {
			return err
		}

		result.ToAccount, err = q.GetAccount(ctx, arg.ToAccountID)
		if err != nil {
			return err
		}

		return nil
	})

	return result, err
}

func AddMoney(
	ctx context.Context,
	q *Queries,
	accountID1 int64,
	amount1 int64,
	accountID2 int64,
	amount2 int64,
) (account1 Account, account2 Account, err error) {
	account1, err = q.AddAccountBalance(ctx, AddAccountBalanceParams{
		ID:     accountID1,
		Amount: amount1,
	})
	if err != nil {
		return
	}

	account2, err = q.AddAccountBalance(ctx, AddAccountBalanceParams{
		ID:     accountID2,
		Amount: amount2,
	})
	if err != nil {
		return
	}

	return
}
