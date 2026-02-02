package logger

import (
	"context"
	"testing"
)

func TestFromContext_WithoutLogger_ReturnsNoop(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	// Must not panic
	l := FromContext(ctx)
	if l == nil {
		t.Fatal("FromContext returned nil, expected no-op logger")
	}

	// Verify it's the noop logger
	if l != noop {
		t.Error("FromContext without logger should return the noop singleton")
	}
}

func TestFromContext_WithLogger_ReturnsSameLogger(t *testing.T) {
	t.Parallel()

	l, err := New("stdout", "stderr")
	if err != nil {
		t.Fatalf("failed to create logger: %v", err)
	}

	ctx := l.WithContext(context.Background())
	got := FromContext(ctx)

	if got == nil {
		t.Fatal("FromContext returned nil")
	}

	// Should not be the noop logger
	if got == noop {
		t.Error("FromContext should return the real logger, not noop")
	}
}

func TestFromContext_ParallelSafety(t *testing.T) {
	t.Parallel()

	l, err := New("stdout", "stderr")
	if err != nil {
		t.Fatalf("failed to create logger: %v", err)
	}

	for i := 0; i < 10; i++ {
		t.Run("", func(t *testing.T) {
			t.Parallel()

			ctx := l.WithContext(context.Background())
			got := FromContext(ctx)
			if got == nil || got == noop {
				t.Error("expected real logger from context")
			}
		})
	}
}

func TestNoopLogger_DoesNotPanic(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	l := FromContext(ctx)

	// Call every method — none should panic
	l.ServerInfo("", "", false)
	l.DBInfo("", "", "", nil, false)
	l.DeliveryInfo(ctx, "", nil)
	l.DeliveryError(ctx, 0, "", nil, nil)
	l.UsecasesInfo("", 0)
	l.UsecasesWarn(nil, 0, nil)
	l.UsecasesError(nil, 0, nil)
	l.RepoInfo("", nil)
	l.RepoWarn(nil, nil)
	l.RepoError(nil, nil)
	l.ExternalInfo(ctx, "", nil)
	l.ExternalWarn(ctx, nil, nil)
	l.ExternalError(ctx, nil, nil)
	l.Sync()
	_ = l.WithContext(ctx)
}
