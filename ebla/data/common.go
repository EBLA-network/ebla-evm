package data

import (
	"os"

	"github.com/EBLA-network/ebla-evm/ebla/util"
	"github.com/EBLA-network/ebla-evm/ebla/util/files"
	"github.com/EBLA-network/ebla-evm/rlp"
)

var this_dir = files.ThisDirRelPath()

func parse_rlp_file(short_file_name string, out interface{}) {
	f, err := os.Open(files.Path(this_dir, short_file_name+".rlp"))
	util.PanicIfNotNil(err)
	util.PanicIfNotNil(rlp.Decode(f, out))
}
