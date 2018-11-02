package main

import (
	"flag"
	"fmt"
	"go/ast"
	"go/build"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"unicode"
)

// TODO: search the entire current package, not just the current file
// TODO: Search for declarations in local scope. (scope.LookupParent?)
// TODO: Support for struct fields.

var debugOn = flag.Bool("d", false, "debug output switch")

func main() {
	flag.Parse()
	if len(flag.Args()) < 2 {
		fmt.Fprintf(os.Stderr, "%s\n", "usage: gotodef <expr> <file>")
		return
	}
	loc, err := findDeclInFile(flag.Args()[0], flag.Args()[1])
	if err != nil {
		debugLog("%v\n", err)
	}
	fmt.Fprintf(os.Stdout, "%s", loc)
}

// findDeclInFile returns the location of the token's declaration, in the form of file:byteOffset.
// It searches the declarations in the supplied file, as well as the imported symbols.
// If the token's declaration is not found, returns an empty string.
func findDeclInFile(tok, srcFname string) (string, error) {
	debugLog("finding %s\n", tok)
	if tok == "" {
		return "", nil
	}
	fset := token.NewFileSet()
	srcF, err := parser.ParseFile(fset, srcFname, nil, 0)
	if err != nil {
		return "", err
	}

	// First, try searching all the declarations in the file specified on the command line.
	if pos := findDecl(tok, srcF.Decls, false); pos != -1 {
		return fset.Position(pos).String(), nil
	}

	// We haven't found the declared symbol in the supplied file.
	// Make sure our search token was exported (first letter capitalized).
	// Then, search the exported symbols from the imported packages for the desired declaration.
	if !unicode.IsUpper(rune(tok[0])) {
		return "", nil
	}
	for _, imp := range srcF.Imports {
		impPath := imp.Path.Value[1 : len(imp.Path.Value)-1]
		srcFpath, err := filepath.Abs(srcFname)
		if err != nil {
			return "", err
		}
		pkg, err := build.Import(impPath, filepath.Dir(srcFpath), 0)
		if err != nil {
			return "", err
		}
		impFnameLists := [][]string{pkg.GoFiles, pkg.CgoFiles}
		for _, impFnames := range impFnameLists {
			for _, impFname := range impFnames {
				impFpath := filepath.Join(pkg.Dir, impFname)
				impF, err := parser.ParseFile(fset, impFpath, nil, 0)
				if err != nil {
					return "", err
				}
				if pos := findDecl(tok, impF.Decls, false); pos != -1 {
					return fset.Position(pos).String(), nil
				}
			}
		}
		/*
			for _, gf := range pkg.CgoFiles {
				fpath := filepath.Join(pkg.Dir, gf)
				impf, err := parser.ParseFile(fset, fpath, nil, 0)
				if err != nil {
					return "", err
				}
				if pos := findDecl(tok, impf.Decls, false); pos != -1 {
					return fset.Position(pos).String(), nil
				}
			}
		*/
		// TODO: Performance: first check scope for the symbol, only parse files on match
		/*
			path := i.Path.Value[1 : len(i.Path.Value)-1] // trim quotes

			// Check the package's exported symbols for our search string.
			debugLog("searching %s's exported symbols\n", path)
			def := importer.Default()
			imp, ok := def.(types.ImporterFrom)
			if !ok {
				log.Fatal("could not cast default Importer to ImporterFrom (not using Go 1.5?")
			}
			pkg, err := imp.ImportFrom(path, ".", 0)
			if err != nil {
				debugLog("%v\n", err)
				return "", -1
			}
			for _, n := range pkg.Scope().Names() {
				debugLog("\thas %s\n", n)
				if n == tok {
					obj := pkg.Scope().Lookup(n)
					if obj == nil {
						debugLog("%v\n", err)
						return "", -1
					}
					bp, err := build.Import(importPath, "", build.FindOnly)
					if err != nil {
						fmt.Fprintf(os.Stderr, "%v\n", err)
						return "", -1
					}
					return pkg.Path(), obj.Pos()
				}
			}
		*/
	}

	return "", nil
}

// test comment delete me
// test comment delete me
// test comment delete me
// test comment delete me
// test comment delete me
// test comment delete me
// test comment delete me
// findDecl searches for the supplied token in the given ast.Decl slice.
// If ignoreImports is true, the decl search will ignore import declarations while traversing the decls.
// TODO: Performance: If we're checking an import's declarations, we can ignore non-exported declarations, avoiding a string compare.
func findDecl(tok string, decls []ast.Decl, ignoreImports bool) token.Pos {
	for _, n := range decls {
		switch t := n.(type) {
		case *ast.FuncDecl:
			debugLog("func decl: %s\n", t.Name.Name)
			if t.Name.Name == tok {
				return t.Name.NamePos
			}
		case *ast.GenDecl:
			switch t.Tok {
			case token.IMPORT:
				if ignoreImports {
					break
				}
				for _, s := range t.Specs {
					i, _ := s.(*ast.ImportSpec)
					noQuotePath := i.Path.Value[1 : len(i.Path.Value)-1]
					// Check the package's name for our search string.
					debugLog("searching package name: %s\n", noQuotePath)
					if i.Name != nil {
						// import name
						debugLog("\timport name: %s\n", i.Name)
						if i.Name.Name == tok {
							return i.Name.NamePos
						}
					} else {
						// import path
						debugLog("\timport path: %s\n", noQuotePath)
						if filepath.Base(noQuotePath) == tok {
							return i.Path.ValuePos
						}
					}
				}
			case token.VAR, token.CONST:
				for _, s := range t.Specs {
					v, _ := s.(*ast.ValueSpec)
					for _, n := range v.Names {
						debugLog("var decl: %s\n", n)
						if n.Name == tok {
							return n.NamePos
						}
					}
				}
			case token.TYPE:
				for _, s := range t.Specs {
					v, _ := s.(*ast.TypeSpec)
					debugLog("type decl: %s\n", v.Name.Name)
					if v.Name.Name == tok {
						return v.Name.NamePos
					}
				}
			}
		}
	}
	return -1
}

func debugLog(format string, v ...interface{}) {
	if !*debugOn {
		return
	}
	fmt.Fprintf(os.Stdout, format, v...)
}
