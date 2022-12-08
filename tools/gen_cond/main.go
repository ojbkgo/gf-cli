package main

import (
	"fmt"
	"os"
	"strconv"

	"github.com/gogf/gf/v2/os/gfile"
	"github.com/gogf/gf/v2/text/gregex"
	"github.com/gogf/gf/v2/text/gstr"
)

func main() {
	st := readStruct()
	fmt.Println(print(st))
}

type arg struct {
	file string
	path string
	line int
}

func args() arg {
	a := arg{}
	a.file = os.Getenv("GOFILE")
	a.path, _ = os.Getwd()
	l := os.Getenv("GOLINE")
	a.line, _ = strconv.Atoi(l)

	return a
}

type stDefine struct {
	Name     string
	FnName   string
	WithPage bool
	Fields   []stField
	DoName   string
}

type stField struct {
	Name string
	Tag  string
	Type string
}

func isTypeArray(t string) bool {
	return gstr.HasPrefix(t, "[]")
}

func isTypeString(t string) bool {
	return t == "string"
}

func print(def stDefine) string {
	s := "func " + def.FnName + "(in *model." + def.Name + ") (*do." + def.DoName + ", []database.Cond) {"

	s += fmt.Sprintf(`
d := &do.%s{}
conds := make([]database.Cond, 0)
`, def.DoName)
	if def.WithPage {
		s += fmt.Sprintf(`
	conds = append(conds, &database.Page{Offset: in.Offset, PageSize: in.PageSize})
`)
	}

	for _, it := range def.Fields {
		if it.Name == "Order" {
			continue
		}
		if isTypeArray(it.Type) {
			s += fmt.Sprintf(`
if len(in.%s) > 0 {
	d.%s = in.%s
}
`, it.Name, it.Name, it.Name)
		} else if isTypeString(it.Type) && it.Name != "OrderBy" {
			s += fmt.Sprintf(`
if len(in.%s) > 0 {
	conds = append(conds, &database.Like{Field: "%s", Val: in.%s})
}
`, it.Name, it.Tag, it.Name)
		} else if it.Name == "OrderBy" {
			s += fmt.Sprintf(`
if len(in.OrderBy) > 0 {

	if in.Order == 1 {
		conds = append(conds, &database.OrderAsc{Field: in.OrderBy})
	} else if in.Order == 2 {
		conds = append(conds, &database.OrderDesc{Field: in.OrderBy})
	}
}
`)
		} else {
			s += fmt.Sprintf(`
if in.%s > 0 {
	d.%s = in.%s
}
`, it.Name, it.Name, it.Name)
		}
	}

	s += `
	return d, conds
}
`

	return s

}

func readStruct() stDefine {
	a := args()

	path := a.path + "/" + a.file
	c := 0
	end := false

	st := stDefine{DoName: os.Args[1]}
	gfile.ReadLines(path, func(text string) error {
		c += 1
		text = gstr.Trim(text, "\n", " ")
		if c <= a.line || end || text == "" {
			return nil
		}

		if c == a.line+1 {
			p := gstr.Split(text, " ")
			st.Name = p[1]
			st.FnName = "get" + gstr.TrimRight(st.Name, "In") + "Cond"
			return nil
		}

		if text == "database.Page" {
			st.WithPage = true
			return nil
		}

		if text == "}" {
			end = true
			return nil
		}
		res, _ := gregex.MatchString(`json:"(\w+)"`, text)
		p := gstr.SplitAndTrim(text, " ", " ", "\n", "\t")
		st.Fields = append(st.Fields, stField{
			Name: p[0],
			Type: p[1],
			Tag:  res[1],
		})

		return nil
	})

	return st
}
