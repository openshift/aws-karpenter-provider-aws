package middleware

import (
	"context"
	"time"

	"github.com/aws/smithy-go/middleware"
	"github.com/jonathan-innis/aws-sdk-go-prometheus/v2/awsmetrics"
)

type StackSerializeEnd struct{}

func GetRecordStackSerializeEndMiddleware() *StackSerializeEnd {
	return &StackSerializeEnd{}
}

func (m *StackSerializeEnd) ID() string {
	return "StackSerializeEnd"
}

func (m *StackSerializeEnd) HandleSerialize(
	ctx context.Context, in middleware.SerializeInput, next middleware.SerializeHandler,
) (
	out middleware.SerializeOutput, metadata middleware.Metadata, err error,
) {

	mctx := awsmetrics.Context(ctx)
	mctx.Data().SerializeEndTime = time.Now().UTC()

	out, metadata, err = next.HandleSerialize(ctx, in)

	return out, metadata, err

}
