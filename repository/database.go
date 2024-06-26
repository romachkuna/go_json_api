package repository

import (
	"fmt"
	"go-api/utilityFunctions"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"log"
	"time"
)

type PostgresDB struct {
	instance *gorm.DB
}

func NewPostgresDB() (*PostgresDB, error) {
	dsn := "host=localhost user=postgres password=romaroma dbname=banking port=5432 sslmode=disable TimeZone=Etc/GMT+4"
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{QueryFields: true})
	if err != nil {
		log.Fatal(err)
	}

	return &PostgresDB{
		instance: db,
	}, nil
}

func (db PostgresDB) GetUserById(id string) (*User, error) {
	var user User
	err := db.instance.Model(&User{}).
		Joins("LEFT JOIN accounts ON users.id = accounts.user_id").
		Joins("LEFT JOIN transactions ON users.id = transactions.user_id").
		Joins("LEFT JOIN responsibilities ON users.id = responsibilities.user_id").
		Where("users.id = ?", id).
		Preload("Accounts").
		Preload("Transactions").
		Preload("Responsibilities").
		First(&user).
		Error

	user.Password = ""
	if err != nil {
		return &user, err
	}
	return &user, nil
}

func (db PostgresDB) LoginUser(id string, password string) bool {
	var count int64
	err := db.instance.Model(&User{}).Where(&User{ID: id, Password: password}).Count(&count).Error
	if err != nil {
		return false
	}
	return count > 0
}
func (db PostgresDB) InsertUser(u *User) error {
	err := db.instance.Create(u).Error
	if err != nil {
		return err
	}
	return nil
}

func (db PostgresDB) InsertAccount(a *Account) (error, string) {
	var accountNumbers []string
	err := db.instance.Model(&Account{}).Pluck("account_number", &accountNumbers).Error
	if err != nil {
		return err, ""
	}
	var accountNumber string
	isUnique := false

	for !isUnique {
		accountNumber = utilityFunctions.GenerateAccountNumber()
		isUnique = !utilityFunctions.Contains(accountNumbers, accountNumber)
	}
	a.AccountNumber = accountNumber
	err = db.instance.Create(a).Error
	if err != nil {
		return err, ""
	}
	return nil, accountNumber
}

func (db PostgresDB) InsertTransaction(t *Transaction) error {
	var transactionIDs []string
	var transactionID string
	err := db.instance.Model(&Transaction{}).Pluck("transaction_id", &transactionIDs).Error
	if err != nil {
		return err
	}
	isUnique := false
	for !isUnique {
		transactionID = utilityFunctions.GenerateTransactionID()
		isUnique = !utilityFunctions.Contains(transactionIDs, transactionID)
	}
	t.TransactionID = transactionID
	// check if the destination account exists
	accountExists, accErr := db.accountExists(t.DestinationAccountNumber)
	if accErr != nil {
		return accErr
	}
	if accountExists {
		destinationAccount, dDbErr := db.getAccountByAccountID(t.DestinationAccountNumber)
		if dDbErr != nil {
			return dDbErr
		}
		originalAccount, oDbErr := db.getAccountByAccountID(t.OriginAccountNumber)
		if oDbErr != nil {
			return oDbErr
		}
		if originalAccount.Status == DISABLED || destinationAccount.Status == DISABLED {
			return fmt.Errorf("SEND OR RECIPIENT'S IS DESIBLED")
		}
		senderBalance := originalAccount.Balance
		switch originalAccount.AccountType {
		case DEBIT:
			if senderBalance >= t.Amount {
				destinationAccount.Balance += t.Amount
				originalAccount.Balance -= t.Amount
				err = db.commitTransaction(t, originalAccount, destinationAccount)
				if err != nil {
					return err
				}
			} else {
				return fmt.Errorf("inssuficient balance")
			}
		case CREDIT:
			destinationAccount.Balance += t.Amount
			originalAccount.Balance -= t.Amount
			err = db.commitTransaction(t, originalAccount, destinationAccount)
			if err != nil {
				return err
			}
		case SAVING:
			return fmt.Errorf("account is not able to make transactions")
		}
	} else {
		return fmt.Errorf("destination Account does not exist")
	}
	return nil
}

func (db PostgresDB) accountExists(accountNumber string) (bool, error) {
	var count int64
	err := db.instance.Model(&Account{}).Where("account_number = ?", accountNumber).Count(&count).Error
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

func (db PostgresDB) getAccountByAccountID(accountNumber string) (*Account, error) {
	var account Account

	err := db.instance.Model(&Account{}).Where("account_number = ?", accountNumber).First(&account).Error
	if err != nil {
		return nil, err
	}
	return &account, nil
}

func (db PostgresDB) AutomaticPayment() {
	go func() {
		var responsibilities []Responsibility
		db.instance.Find(&responsibilities)
		for _, responsibility := range responsibilities {
			go db.processResponsibility(responsibility)
		}
	}()
}

func (db PostgresDB) processResponsibility(r Responsibility) {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if time.Now().After(r.DueDate) {
				t := Transaction{
					OriginAccountNumber:      r.OriginAccountNumber,
					DestinationAccountNumber: r.DestinationAccountNumber,
					Type:                     SENT,
					Date:                     time.Now(),
					Amount:                   r.Amount,
					Description:              r.Description,
					TransactionID:            "",
					UserID:                   r.UserID,
				}

				err := db.InsertTransaction(&t)
				if err != nil {
					fmt.Println(err)
					return
				}

				return
			}
		}
	}
}

func (db PostgresDB) commitTransaction(t *Transaction, originalAccount *Account, destinationAccount *Account) error {
	transactionERR := db.instance.Transaction(func(tx *gorm.DB) error {
		updateDestinationAccount := db.instance.Clauses(clause.Locking{Strength: "UPDATE"}).Save(&destinationAccount).Error
		if updateDestinationAccount != nil {
			return updateDestinationAccount
		}
		updateOriginalAccount := db.instance.Clauses(clause.Locking{Strength: "UPDATE"}).Save(&originalAccount).Error
		if updateOriginalAccount != nil {
			return updateOriginalAccount
		}
		addTransaction := db.instance.Create(t).Error
		if addTransaction != nil {
			return addTransaction
		}
		return nil
	})
	if transactionERR != nil {
		return transactionERR
	}
	return nil
}
