package main

import (
	"bufio"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
)

type symbol int

const (
	SOI symbol = 0xd8
	EOI symbol = 0xd9
)

func (s symbol) Short() string {
	switch s {
	case SOI:
		return "SOI"
	case EOI:
		return "EOI"
	case 0xc4:
		return "DHT"
	case 0xdb:
		return "DQT"
	case 0xda:
		return "SOS"
	case 0xdd:
		return "DRI"
	case 0xfe:
		return "COM"
	}
	switch {
	case 0xc0 <= s && s <= 0xcf:
		return fmt.Sprintf("SOF%d", s-0xc0)
	case 0xd0 <= s && s <= 0xd7:
		return fmt.Sprintf("RST%d", s-0xd0)
	case 0xe0 <= s && s <= 0xef:
		return fmt.Sprintf("APP%d", s-0xe0)
	}
	return fmt.Sprintf("UNK%#x", int(s))

}

func (s symbol) Long() string {
	switch s {
	case SOI:
		return "Start Of Image."
	case EOI:
		return "End Of Image."
	case 0xc0:
		return "Start Of Frame (Baseline)."
	case 0xc2:
		return "Start Of Frame (Progressive)."
	case 0xc4:
		return "Define Huffman Table."
	case 0xdb:
		return "Define Quantization Table."
	case 0xda:
		return "Start Of Scan."
	case 0xdd:
		return "Define Restart Interval."
	case 0xfe:
		return "COMment."
	}
	switch {
	case 0xd0 <= s && s <= 0xd7:
		return fmt.Sprintf("ReSTart (%d).", s-0xd0)
	case 0xe0 <= s && s <= 0xef:
		return fmt.Sprintf("APPlication specific (%d).", s-0xe0)
	}
	return fmt.Sprintf("Unknown symbol: %#x", int(s))

}

var ErrNotJpeg = errors.New("missing jpeg magic")

type marker struct {
	sym    symbol
	offset int
	size   int
}

type config struct {
	showOffset bool
	showSize   bool
	hex        bool
}

type Reader interface {
	io.ByteReader
	io.Reader
}

func printInfo(file string, r Reader, c config) error {
	var (
		offset  int
		lastb   byte
		markers []marker
	)
	defer func() {
		for _, m := range markers {
			fmt.Printf("%s:%s", file, m.sym.Short())
			if c.showOffset {
				if c.hex {
					fmt.Printf(":%#x", m.offset-2)
				} else {
					fmt.Printf(":%#d", m.offset-2)
				}
			}
			if c.showSize {
				if c.hex {
					fmt.Printf(":%#x", m.size)
				} else {
					fmt.Printf(":%#d", m.size)
				}
			}
			fmt.Println()
		}
	}()
	for {
		b, err := r.ReadByte()
		if err != nil {
			return err
		}
		offset++
		if lastb == 0xff && b != 0xff && b != 0 {
			p := make([]byte, 2)
			sym := symbol(b)
			m := marker{
				offset: offset,
				sym:    sym,
			}
			if sym != EOI && sym != SOI {
				_, err := io.ReadFull(r, p)
				if err != nil {
					return err
				}
				offset += 2
				m.size = int(p[0])<<8 + int(p[1])
			}

			markers = append(markers, m)
			if sym == 0xda { // SOS
				dumpSOS(r, m.size)
			}
		}
		lastb = b
	}
}

func dumpSOS(r Reader, n int) {
	var tmp [16]byte
	_, _ = r.Read(tmp[:n])
	ncomp := int(tmp[0])
	ss := tmp[1+2*ncomp]
	se := tmp[2+2*ncomp]
	a := tmp[3+2*ncomp]
	ah := a >> 4
	al := a & 0xf
	fmt.Printf("SOS\tss=%d\tse=%d\tah=%d\tal=%d\n", ss, se, ah, al)
	for i := 0; i < ncomp; i++ {
		fmt.Printf("  #%d", tmp[2*i+1])
		td := tmp[2*i+2] >> 4
		ta := tmp[2*i+2] & 0xf
		fmt.Printf(" td=%d ta=%d", td, ta)
		fmt.Printf("\n")
	}
}

func main() {
	var c config
	flag.BoolVar(&c.showOffset, "offset", false, "show offset each marker was found at.")
	flag.BoolVar(&c.showSize, "size", false, "show size from header of each marker.")
	flag.BoolVar(&c.hex, "hex", false, "show size and offset in hex.")

	flag.Parse()
	for _, file := range flag.Args() {
		f, err := os.Open(file)
		if err != nil {
			log.Println(err)
			continue
		}

		r := bufio.NewReader(f)
		if err := printInfo(file, r, c); err != nil && err != io.EOF {
			log.Fatal(file, err)
		}
		f.Close()
	}
}
