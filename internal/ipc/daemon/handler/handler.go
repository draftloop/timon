package handler

type Response[T any] struct {
	Data        *T
	ClientError error
	DaemonError error
}

func (r Response[T]) Send(response T) Response[T] {
	return Response[T]{Data: &response}
}

func (r Response[T]) SendClientError(err error) Response[T] {
	return Response[T]{ClientError: err}
}

func (r Response[T]) SendDaemonError(err error) Response[T] {
	return Response[T]{DaemonError: err}
}
