package css

import (
	"fmt"
	"testing"
)

func TestPrintClassNamesAsGoConstants(t *testing.T) {
	tailwind, err := DownloadTailwind()
	if err != nil {
		t.Fatal()
	}

	if err := PrintClassNamesAsGoConstants(tailwind); err != nil {
		t.Fatal(err)
	}
}

func Test_text2GoIdentifier(t *testing.T) {
	fmt.Println(text2GoIdentifier("32xl:absolute"))
}