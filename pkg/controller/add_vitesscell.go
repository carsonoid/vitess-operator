package controller

import (
	"vitess.io/vitess-operator/pkg/controller/vitesscell"
)

func init() {
	// AddToManagerFuncs is a list of functions to create controllers and add them to a manager.
	AddToManagerFuncs = append(AddToManagerFuncs, vitesscell.Add)
}
