package tenant

import "context"

type workspaceKey struct{}

// WithWorkspace stores the tenant workspace path in context.
func WithWorkspace(ctx context.Context, workspace string) context.Context {
	return context.WithValue(ctx, workspaceKey{}, workspace)
}

// WorkspaceFromContext retrieves the tenant workspace path from context.
func WorkspaceFromContext(ctx context.Context) (string, bool) {
	workspace, ok := ctx.Value(workspaceKey{}).(string)
	return workspace, ok
}
