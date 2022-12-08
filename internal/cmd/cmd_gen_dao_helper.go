package cmd

import (
	"context"
	"os"
	"strings"

	_ "github.com/denisenkom/go-mssqldb"
	"github.com/gogf/gf/v2/database/gdb"
	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/os/gfile"
	"github.com/gogf/gf/v2/text/gregex"
	"github.com/gogf/gf/v2/text/gstr"
	_ "github.com/lib/pq"
	//_ "github.com/mattn/go-oci8"
	//_ "github.com/mattn/go-sqlite3"

	"github.com/gogf/gf-cli/v2/utility/mlog"
	"github.com/gogf/gf-cli/v2/utility/utils"
)

func generateEntityHelper(ctx context.Context, db gdb.DB, tableNames, newTablesName []string, in cGenDaoInternalInput) {
	if !in.cGenDaoInput.WithHelper {
		return
	}

	var (
		helperDirPath = gfile.Join(in.Path, defaultHelperPath)
		importPrefix  = in.ImportPrefix
		dirRealPath   = gfile.RealPath(in.Path)
		realPrefix    = in.ImportPrefix
	)

	if importPrefix == "" {
		if dirRealPath == "" {
			dirRealPath = in.Path
			importPrefix = dirRealPath
			importPrefix = gstr.Trim(dirRealPath, "./")
		} else {
			importPrefix = gstr.Replace(dirRealPath, gfile.Pwd(), "")
		}
		importPrefix = gstr.Replace(importPrefix, gfile.Separator, "/")

		realPrefix = gstr.Join(g.SliceStr{in.ModName, importPrefix}, "/")
		realPrefix, _ = gregex.ReplaceString(`\/{2,}`, `/`, gstr.Trim(realPrefix, "/"))

		importPrefix = gstr.Join(g.SliceStr{in.ModName, importPrefix, defaultEntityPath}, "/")
		importPrefix, _ = gregex.ReplaceString(`\/{2,}`, `/`, gstr.Trim(importPrefix, "/"))
	}

	if os.Getenv("IS_CHAITIN") == "yes" {
		importPrefix = strings.Replace(importPrefix, "git.in.chaitin.net/veinmind/backend/submodule/backend/", "github.com/chaitin/veinmind-backend/", -1)
		realPrefix = strings.Replace(realPrefix, "git.in.chaitin.net/veinmind/backend/submodule/backend/", "github.com/chaitin/veinmind-backend/", -1)
	}

	for i, tableName := range tableNames {
		fieldMap, err := db.TableFields(ctx, tableName)
		if err != nil {
			mlog.Fatalf("fetching tables fields failed for table '%s':\n%v", in.TableName, err)
		}

		var (
			newTableName   = newTablesName[i]
			helperFilePath = gfile.Join(helperDirPath, gstr.CaseSnake(newTableName)+".go")
			helperContent  = generateHelperContent(realPrefix, gstr.CaseCamel(newTableName), importPrefix, fieldMap)
		)
		if len(helperContent) == 0 {
			return
		}
		err = gfile.PutContents(helperFilePath, strings.TrimSpace(helperContent))
		if err != nil {
			mlog.Fatalf("writing content to '%s' failed: %v", helperFilePath, err)
		} else {
			utils.GoFmt(helperFilePath)
			mlog.Print("generated:", helperFilePath)
		}
	}
}

var (
	// ImportPkg,StructName
	structTpl = `
package helper

import (
	"context"

	"{ImportPkg}"
	"{ImportPkgPrefix}/service/do"
	"{ImportPkgPrefix}/service/dao"
	"github.com/chaitin/veinmind-backend/utility/cond/database"
)


type {StructName}List struct {
		List []entity.{StructName}
}

func New{StructName}List(l []entity.{StructName}) *{StructName}List {
	return &{StructName}List{
		List: l,
	}
}

func (l *{StructName}List) FetchDB(ctx context.Context, fields []interface{}, wd *do.{StructName}, cond ...database.Cond) error {
	es := make([]entity.{StructName}, 0)
	if err := dao.{StructName}.GetList(ctx, &es, fields, wd, cond...); err != nil {
		return err
	}

	l.List = es
	
	return nil
}

func (l *{StructName}List) Empty() bool {
	return len(l.List) == 0
}

func (l *{StructName}List) All() []entity.{StructName} {
	return l.List
}

func (l *{StructName}List) Count() int {
	return len(l.List)
}

func (l *{StructName}List) Foreach(fn func(int, *entity.{StructName})) {
	for i := range l.List {
		fn(i, &l.List[i])
	}
}

func (l *{StructName}List) GetByIndex(idx int) *entity.{StructName} {
	if idx > len(l.List) {
		return nil
	}

	v := l.List[idx]

	return &v
}

func (l *{StructName}List) BatchGetByIndex(idxList []int) []*entity.{StructName} {
	res := make([]*entity.{StructName}, 0, len(idxList))

	for i := range idxList {
		res = append(res, &l.List[i])
	}

	return res
}
`

	// StructName, FieldNameCaseCamel, FieldType
	indexByTpl = `
func (l *{StructName}List) IndexBy{FieldNameCaseCamel}() map[{FieldType}]int {
	if len(l.List) == 0 {
		return nil
	}
	res := make(map[{FieldType}]int)

	for i := range l.List {
		res[l.List[i].{FieldNameCaseCamel}] = i
	}

	return res
}

`
	// StructName,FieldNameCaseCamel,FieldType,
	valTpl = `
func (l *{StructName}List) {FieldNameCaseCamel}Vals() []{FieldType} {
	if len(l.List) == 0 {
		return nil
	}
	res := make([]{FieldType}, 0, len(l.List))

	for i := range l.List {
		res = append(res, l.List[i].{FieldNameCaseCamel})
	}

	return res
}
`
	// StructName, FieldNameCaseCamel, FieldType
	groupByTpl = `
func (l *{StructName}List) GroupBy{FieldNameCaseCamel}() map[{FieldType}][]int {
	if len(l.List) == 0 {
		return nil
	}

	res := make(map[{FieldType}][]int)

	for i := range l.List {
		if _, ok := res[l.List[i].{FieldNameCaseCamel}]; !ok {
			res[l.List[i].{FieldNameCaseCamel}] = make([]int, 0)
		}

		res[l.List[i].{FieldNameCaseCamel}] = append(res[l.List[i].{FieldNameCaseCamel}], i)
	}

	return res
}
`
)

func generateHelperContent(moduelpkg, structName, entPkgName string, fieldsMap map[string]*gdb.TableField) string {
	res := ""

	for k := range fieldsMap {
		f := fieldsMap[k]

		if gstr.Contains(f.Comment, "val") {
			res += gstr.ReplaceByMap(valTpl, g.MapStrStr{
				"{StructName}":         structName,
				"{FieldNameCaseCamel}": gstr.CaseCamel(f.Name),
				"{FieldType}":          getGolangType(f, false, false),
			})

			res += "\n"
		}

		if gstr.Contains(f.Comment, "index") {
			res += gstr.ReplaceByMap(indexByTpl, g.MapStrStr{
				"{StructName}":         structName,
				"{FieldNameCaseCamel}": gstr.CaseCamel(f.Name),
				"{FieldType}":          getGolangType(f, false, false),
			})

			res += "\n"
		}

		if gstr.Contains(f.Comment, "group") {
			res += gstr.ReplaceByMap(groupByTpl, g.MapStrStr{
				"{StructName}":         structName,
				"{FieldNameCaseCamel}": gstr.CaseCamel(f.Name),
				"{FieldType}":          getGolangType(f, false, false),
			})

			res += "\n"
		}
	}

	res = gstr.ReplaceByMap(structTpl, g.MapStrStr{
		"{StructName}":      structName,
		"{ImportPkg}":       entPkgName,
		"{ImportPkgPrefix}": moduelpkg,
	}) + "\n" + res

	return res
}
