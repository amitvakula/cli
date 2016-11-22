package api

import (
	"errors"
	"strconv"
)

func FindPermissionById(id string, perms []*Permission) *Permission {
	for _, x := range perms {
		if x.Id == id {
			return x
		}
	}
	return &Permission{
		Id:    id,
		Level: "none",
	}
}

// coalesce will extract an API error message into a golang error, if applicable.
func coalesce(err error, aerr *ApiError) error {
	if err != nil {
		return err
	} else if aerr != nil {
		if aerr.Message == "" {
			aerr.Message = "Unknown server error"
		}
		aerr.Message = "(" + strconv.Itoa(aerr.StatusCode) + ") " + aerr.Message
		return errors.New(aerr.Message)
	} else {
		return nil
	}
}
