package types

import "io"

type RenderCtx func(w io.Writer, data RenderData, names ...string) error

type RenderData map[string]interface{}
