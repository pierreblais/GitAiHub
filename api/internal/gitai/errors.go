package gitai

import (
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func errNotFound(what string) error { return status.Error(codes.NotFound, what+" not found") }
func errInvalid(msg string) error   { return status.Error(codes.InvalidArgument, msg) }
