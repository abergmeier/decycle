package internal

import (
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/go-python/gpython/ast"
	"github.com/go-python/gpython/parser"
)

type processedKey struct {
	Lineno    int
	ColOffset int
}

func ParseFile(filename string) ([]Import, error) {
	f, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	a, err := parser.Parse(f, filename, "exec")
	if err != nil {
		return nil, err
	}

	m := a.(*ast.Module)
	ids := findImports(m)
	return ids, nil
}

type fileImport struct {
	Filename string
	Imps     []Import
}

func ParseDir(dir string) <-chan fileImport {
	c := make(chan fileImport)
	go func() {
		defer close(c)
		wg := sync.WaitGroup{}

		err := filepath.Walk(dir,
			func(path string, info os.FileInfo, err error) error {
				if err != nil {
					return err
				}
				if !strings.HasSuffix(path, ".py") {
					return nil
				}

				wg.Add(1)
				go func() {
					defer wg.Done()
					imps, err := ParseFile(path)
					if err != nil {
						panic(err)
					}
					c <- fileImport{
						Filename: path,
						Imps:     imps,
					}
				}()

				return nil
			})
		wg.Wait()
		if err != nil {
			panic(err)
		}
	}()
	return c
}

type Import struct {
	Name      ast.Identifier
	Lineno    int
	ColOffset int
}

func findImports(m *ast.Module) []Import {

	toBeProcessed := []ast.Stmt{}
	processed := map[processedKey]bool{}
	toBeProcessed = append(toBeProcessed, m.Body...)

	imported := []Import{}

	for len(toBeProcessed) != 0 {

		// Since we always only append to the slice
		// we can reverse iterate without any problems
		for i := len(toBeProcessed); i > 0; i-- {
			stmt := toBeProcessed[i-1]
			toBeProcessed = popFromList(toBeProcessed)

			key := processedKey{
				Lineno:    stmt.GetLineno(),
				ColOffset: stmt.GetColOffset(),
			}
			_, ok := processed[key]
			if ok {
				continue
			}
			processed[key] = true
			switch s := stmt.(type) {
			case *ast.Assign:
				continue
			case *ast.Assert:
				continue
			case *ast.AugAssign:
				continue
			case *ast.Delete:
				continue
			case *ast.Pass:
				continue
			case *ast.ClassDef:
				toBeProcessed = append(toBeProcessed, s.Body...)
			case *ast.ExprStmt:
				// TODO: Check whether we really want to ignore
				continue
			case *ast.For:
				toBeProcessed = append(toBeProcessed, s.Body...)
				toBeProcessed = append(toBeProcessed, s.Orelse...)
			case *ast.FunctionDef:
				toBeProcessed = append(toBeProcessed, s.Body...)
			case *ast.Global:
				continue
			case *ast.If:
				toBeProcessed = append(toBeProcessed, s.Body...)
				toBeProcessed = append(toBeProcessed, s.Orelse...)
			case *ast.Import:
				for _, a := range s.Names {
					imported = append(imported, Import{
						Name:      a.Name,
						Lineno:    a.Lineno,
						ColOffset: a.ColOffset,
					})
				}
			case *ast.ImportFrom:
				imported = append(imported, Import{
					Name:      s.Module,
					Lineno:    s.Lineno,
					ColOffset: s.ColOffset,
				})
			case *ast.Return:
				continue
			case *ast.Try:
				toBeProcessed = append(toBeProcessed, s.Body...)
				toBeProcessed = append(toBeProcessed, s.Orelse...)
				toBeProcessed = append(toBeProcessed, s.Finalbody...)
			case *ast.While:
				toBeProcessed = append(toBeProcessed, s.Body...)
				toBeProcessed = append(toBeProcessed, s.Orelse...)
			case *ast.With:
				toBeProcessed = append(toBeProcessed, s.Body...)
			default:
				panic(s)
			}
		}
	}

	return imported
}

func popFromList(s []ast.Stmt) []ast.Stmt {
	return s[:len(s)-1]
}
