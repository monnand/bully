package main

import (
	"math/big"
	"crypto/sha256"
	"github.com/nu7hatch/gouuid"
)

func stringToBig(str string) *big.Int {
	sha := sha256.New()
	sha.Write([]byte(str))
	b := sha.Sum(nil)

	ret := new(big.Int).SetBytes(b)
	return ret
}

type Bully struct {
	myId *big.Int
	myIdStr string
}

func NewBully(myid string) *Bully {
	if len(myid) == 0 {
		uu, _ := uuid.NewV4()
		myid = uu.String()
	}
	ret := new(Bully)
	ret.myId = stringToBig(myid)
	ret.myIdStr = myid
	return ret
}

