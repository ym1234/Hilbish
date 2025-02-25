package main

import (
	"strings"
	"sync"

	"hilbish/util"

	rt "github.com/arnodel/golua/runtime"
)

var aliases *aliasHandler

type aliasHandler struct {
	aliases map[string]string
	mu *sync.RWMutex
}

// initialize aliases map
func newAliases() *aliasHandler {
	return &aliasHandler{
		aliases: make(map[string]string),
		mu: &sync.RWMutex{},
	}
}

func (a *aliasHandler) Add(alias, cmd string) {
	a.mu.Lock()
	defer a.mu.Unlock()

	a.aliases[alias] = cmd
}

func (a *aliasHandler) All() map[string]string {
	return a.aliases
}

func (a *aliasHandler) Delete(alias string) {
	a.mu.Lock()
	defer a.mu.Unlock()

	delete(a.aliases, alias)
}

func (a *aliasHandler) Resolve(cmdstr string) string {
	a.mu.RLock()
	defer a.mu.RUnlock()

	args := strings.Split(cmdstr, " ")
	for a.aliases[args[0]] != "" {
		alias := a.aliases[args[0]]
		cmdstr = alias + strings.TrimPrefix(cmdstr, args[0])
		cmdArgs, _ := splitInput(cmdstr)
		args = cmdArgs

		if a.aliases[args[0]] == alias {
			break
		}
		if a.aliases[args[0]] != "" {
			continue
		}
	}

	return cmdstr
}

// lua section

func (a *aliasHandler) Loader(rtm *rt.Runtime) *rt.Table {
	// create a lua module with our functions
	hshaliasesLua := map[string]util.LuaExport{
		"add": util.LuaExport{hlalias, 2, false},
		"list": util.LuaExport{a.luaList, 0, false},
		"del": util.LuaExport{a.luaDelete, 1, false},
	}

	mod := rt.NewTable()
	util.SetExports(rtm, mod, hshaliasesLua)

	return mod
}

func (a *aliasHandler) luaList(t *rt.Thread, c *rt.GoCont) (rt.Cont, error) {
	aliasesList := rt.NewTable()
	for k, v := range a.All() {
		aliasesList.Set(rt.StringValue(k), rt.StringValue(v))
	}

	return c.PushingNext1(t.Runtime, rt.TableValue(aliasesList)), nil
}

func (a *aliasHandler) luaDelete(t *rt.Thread, c *rt.GoCont) (rt.Cont, error) {
	if err := c.Check1Arg(); err != nil {
		return nil, err
	}
	alias, err := c.StringArg(0)
	if err != nil {
		return nil, err
	}
	a.Delete(alias)

	return c.Next(), nil
}
