package controller

import (
	"vitess.io/vitess-operator/pkg/controller/vitessshard"
)

func init() {
	// AddToManagerFuncs is a list of functions to create controllers and add them to a manager.
	AddToManagerFuncs = append(AddToManagerFuncs, vitessshard.Add)
}
