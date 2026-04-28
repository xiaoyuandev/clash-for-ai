package localgateway

import "context"

type Service interface {
	Handle(ctx context.Context, request Request, source ModelSource) (Response, error)
}
