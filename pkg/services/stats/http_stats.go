package stats

import (
	"github.com/jamiealquiza/tachymeter"
	"sync/atomic"
)

type RequestStats struct {
	Success atomic.Int32
	Fail    atomic.Int32
	Meter   *tachymeter.Tachymeter
}
