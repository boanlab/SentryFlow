package common

import (
	"github.com/google/uuid"
	"time"
)

type Log struct {
	TimeStamp time.Time `bson:"timeStamp"`
	UUID      uuid.UUID `bson:"uuid"`
	Method    string    `bson:"method"`
	Path      string    `bson:"path"`
	Protocol  string    `bson:"protocol"`
	SrcIP     string    `bson:"srcip"`
	SrcName   string    `bson:"srcname"`
	DstIP     string    `bson:"dstip"`
	DstName   string    `bson:"dstname"`
}
