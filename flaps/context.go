package flaps

import "context"

type (
	machineIDCtxKey struct{}
	actionCtxKey    struct{}
)

func contextWithMachineID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, machineIDCtxKey{}, id)
}

func machineIDFromContext(ctx context.Context) string {
	value := ctx.Value(machineIDCtxKey{})
	if value == nil {
		return ""
	}
	return value.(string)
}

func contextWithAction(ctx context.Context, action flapsAction) context.Context {
	return context.WithValue(ctx, actionCtxKey{}, action)
}

func actionFromContext(ctx context.Context) flapsAction {
	value := ctx.Value(actionCtxKey{})
	if value == nil {
		return none
	}
	return value.(flapsAction)
}
