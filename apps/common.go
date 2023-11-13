package apps

import "io"

type IOs struct {
	In  io.Reader
	Out io.Writer
	Err io.Writer
}
