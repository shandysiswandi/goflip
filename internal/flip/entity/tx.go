package entity

type Transaction struct {
	Timestamp    int64
	Counterparty string
	Type         TxType
	Amount       int64
	Status       TxStatus
	Description  string
}
