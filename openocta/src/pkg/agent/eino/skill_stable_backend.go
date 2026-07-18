package eino

import (
	"context"

	"github.com/cloudwego/eino/adk/middlewares/skill"
)

// stableSkillBackend wraps a skill backend so metadata reads are not aborted by
// a canceled agent turn context (e.g. preempt/abort while ToolsNode builds tool info).
type stableSkillBackend struct {
	inner skill.Backend
}

func (b stableSkillBackend) List(ctx context.Context) ([]skill.FrontMatter, error) {
	return b.inner.List(context.WithoutCancel(ctx))
}

func (b stableSkillBackend) Get(ctx context.Context, name string) (skill.Skill, error) {
	return b.inner.Get(context.WithoutCancel(ctx), name)
}
