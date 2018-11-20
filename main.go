package main

import (
	"github.com/sirupsen/logrus"

	"github.com/chanzuckerberg/blessclient/cmd"
)

func main() {
	if err := cmd.Execute(); err != nil {
		logrus.Fatal(err)
	}
}
