package interpreter

import (
	"elara/lexer"
	"elara/parser"
	"reflect"
)

type Command interface {
	Exec(ctx *Context) *Value
}

type DefineVarCommand struct {
	Name    string
	Mutable bool
	Type    Type
	value   Command
}

func (c DefineVarCommand) Exec(ctx *Context) *Value {
	value := c.value.Exec(ctx)
	if !c.Type.Accepts(*value.Type) {
		panic("Cannot use value of type " + value.Type.Name + " in place of " + c.Type.Name + " for variable " + c.Name)
	}
	variable := Variable{
		Name:    c.Name,
		Mutable: c.Mutable,
		Type:    c.Type,
		Value:   value,
	}

	ctx.DefineVariable(c.Name, variable)
	return nil
}

type AssignmentCommand struct {
	Name  string
	value Command
}

func (c *AssignmentCommand) Exec(ctx *Context) *Value {
	variable := ctx.FindVariable(c.Name)
	if variable == nil {
		panic("No such variable " + c.Name)
	}

	if !variable.Mutable {
		panic("Cannot reassign immutable variable " + c.Name)
	}

	value := c.value.Exec(ctx)

	if !variable.Type.Accepts(*value.Type) {
		panic("Cannot reassign variable " + c.Name + " of type " + variable.Type.Name + " to value " + *value.String() + " of type " + value.Type.Name)
	}

	variable.Value = value
	return nil
}

type VariableCommand struct {
	Variable string
}

func (c *VariableCommand) Exec(ctx *Context) *Value {
	variable := ctx.FindVariable(c.Variable)
	if variable == nil {
		param := ctx.FindParameter(c.Variable)
		if param == nil {
			panic("No such variable or parameter " + c.Variable)
		}
		return param
	}
	return variable.Value
}

type InvocationCommand struct {
	Invoking Command
	args     []Command
}

func (c *InvocationCommand) Exec(ctx *Context) *Value {
	context, isContext := c.Invoking.(*ContextCommand)

	val := c.Invoking.Exec(ctx)
	fun, ok := val.Value.(Function)
	if !ok {
		panic("Cannot invoke non-function")
	}
	if !isContext {
		return fun.Exec(ctx, nil, c.args)
	}

	//ContextCommand seems to think it's a special case... because it is.
	receiver := context.receiver.Exec(ctx)
	function, ok := receiver.Type.functions[context.variable]
	if !ok {
		panic("No such variable " + context.variable + " on type " + receiver.Type.Name)
	}

	exec := context.receiver.Exec(ctx)
	return function.Exec(ctx, exec, c.args)

}

type AbstractCommand struct {
	content func(ctx *Context) *Value
}

func (c *AbstractCommand) Exec(ctx *Context) *Value {
	return c.content(ctx)
}

func NewAbstractCommand(content func(ctx *Context) *Value) *AbstractCommand {
	return &AbstractCommand{
		content: content,
	}
}

type LiteralCommand struct {
	value Value
}

func (c *LiteralCommand) Exec(_ *Context) *Value {
	return &c.value
}

type BinaryOperatorCommand struct {
	lhs Command
	op  func(ctx *Context, lhs *Value, rhs *Value) *Value
	rhs Command
}

func (c *BinaryOperatorCommand) Exec(ctx *Context) *Value {
	lhs := c.lhs.Exec(ctx)
	rhs := c.rhs.Exec(ctx)

	return c.op(ctx, lhs, rhs)
}

type BlockCommand struct {
	lines []Command
}

func (c *BlockCommand) Exec(ctx *Context) *Value {
	var last *Value
	for _, line := range c.lines {
		last = line.Exec(ctx)
	}
	return last
}

type ContextCommand struct {
	receiver Command
	variable string
}

func (c *ContextCommand) Exec(ctx *Context) *Value {
	receiver := c.receiver.Exec(ctx)
	function, ok := receiver.Type.functions[c.variable]
	if !ok {
		panic("No such variable " + c.variable + " on type " + receiver.Type.Name)
	}

	return &Value{
		Type:  FunctionType(&c.variable, function),
		Value: function,
	}
}

type IfElseCommand struct {
	condition  Command
	ifBranch   Command
	elseBranch Command
}

func (c *IfElseCommand) Exec(ctx *Context) *Value {
	condition := c.condition.Exec(ctx)
	asBoolean, ok := condition.Value.(bool)
	if !ok {
		panic("If statement requires boolean value")
	}
	if asBoolean {
		return c.ifBranch.Exec(ctx)
	} else if c.elseBranch != nil {
		return c.elseBranch.Exec(ctx)
	} else {
		return nil
	}
}

func ToCommand(statement parser.Stmt) Command {

	switch t := statement.(type) {
	case parser.VarDefStmt:
		Type := FromASTType(t.Type)
		if Type == nil {
			Type = AnyType
		}
		valueExpr := ExpressionToCommand(t.Value)
		return DefineVarCommand{
			Name:    t.Identifier,
			Mutable: t.Mutable,
			Type:    *Type,
			value:   valueExpr,
		}
	case parser.ExpressionStmt:
		return ExpressionToCommand(t.Expr)

	case parser.BlockStmt:
		commands := make([]Command, len(t.Stmts))
		for i, stmt := range t.Stmts {
			commands[i] = ToCommand(stmt)
		}
		return &BlockCommand{lines: commands}

	case parser.IfElseStmt:
		condition := ExpressionToCommand(t.Condition)
		ifBranch := ToCommand(t.MainBranch)
		var elseBranch Command
		if t.ElseBranch != nil {
			elseBranch = ToCommand(t.ElseBranch)
		}

		return &IfElseCommand{
			condition:  condition,
			ifBranch:   ifBranch,
			elseBranch: elseBranch,
		}
	}

	panic("Could not handle " + reflect.TypeOf(statement).Name())
}

func ExpressionToCommand(expr parser.Expr) Command {

	switch t := expr.(type) {
	case parser.VariableExpr:
		return &VariableCommand{Variable: t.Identifier}

	case parser.InvocationExpr:
		fun := ExpressionToCommand(t.Invoker)
		args := make([]Command, 0)
		for _, arg := range t.Args {
			command := ExpressionToCommand(arg)
			if command == nil {
				panic("Could not convert expression " + reflect.TypeOf(arg).Name() + " to command")
			}
			args = append(args, command)
		}

		return &InvocationCommand{
			Invoking: fun,
			args:     args,
		}

	case parser.StringLiteralExpr:
		str := t.Value
		value := Value{
			Type:  StringType,
			Value: str,
		}
		return &LiteralCommand{value: value}

	case parser.IntegerLiteralExpr:
		integer := t.Value
		value := Value{
			Type:  IntType,
			Value: integer,
		}
		return &LiteralCommand{value: value}
	case parser.FloatLiteralExpr:
		float := t.Value
		value := Value{
			Type:  FloatType,
			Value: float,
		}
		return &LiteralCommand{value: value}
	case parser.BooleanLiteralExpr:
		boolean := t.Value
		value := Value{
			Type:  BooleanType,
			Value: boolean,
		}
		return &LiteralCommand{value: value}

	case parser.BinaryExpr:
		lhs := t.Lhs
		lhsCmd := ExpressionToCommand(lhs)
		op := t.Op
		rhs := t.Rhs
		rhsCmd := ExpressionToCommand(rhs)

		switch op {
		case lexer.Add:
			return &BinaryOperatorCommand{
				lhs: lhsCmd,
				op: func(ctx *Context, lhs *Value, rhs *Value) *Value {
					left := lhs.Value
					lhsInt, ok := left.(int64)
					if !ok {
						panic("LHS must be an int64")
					}
					rhsInt, ok := rhs.Value.(int64)
					if !ok {
						panic("RHS must be an int64")
					}

					return &Value{
						Type:  IntType,
						Value: lhsInt + rhsInt,
					}
				},
				rhs: rhsCmd,
			}
		case lexer.Subtract:
			return &BinaryOperatorCommand{
				lhs: lhsCmd,
				op: func(ctx *Context, lhs *Value, rhs *Value) *Value {
					left := lhs.Value
					lhsInt, ok := left.(int64)
					if !ok {
						panic("LHS must be an int64")
					}
					rhsInt, ok := rhs.Value.(int64)
					if !ok {
						panic("RHS must be an int64")
					}

					return &Value{
						Type:  IntType,
						Value: lhsInt - rhsInt,
					}
				},
				rhs: rhsCmd,
			}
		case lexer.Multiply:
			return &BinaryOperatorCommand{
				lhs: lhsCmd,
				op: func(ctx *Context, lhs *Value, rhs *Value) *Value {
					left := lhs.Value
					lhsInt, ok := left.(int64)
					if !ok {
						panic("LHS must be an int64")
					}
					rhsInt, ok := rhs.Value.(int64)
					if !ok {
						panic("RHS must be an int64")
					}

					return &Value{
						Type:  IntType,
						Value: lhsInt * rhsInt,
					}
				},
				rhs: rhsCmd,
			}
		case lexer.Slash:
			return &BinaryOperatorCommand{
				lhs: lhsCmd,
				op: func(ctx *Context, lhs *Value, rhs *Value) *Value {
					left := lhs.Value
					lhsInt, ok := left.(int64)
					if !ok {
						panic("LHS must be an int64")
					}
					rhsInt, ok := rhs.Value.(int64)
					if !ok {
						panic("RHS must be an int64")
					}

					return &Value{
						Type:  IntType,
						Value: lhsInt / rhsInt,
					}
				},
				rhs: rhsCmd,
			}
		}
	case parser.FuncDefExpr:
		params := make([]Parameter, len(t.Arguments))
		for i, parameter := range t.Arguments {
			paramType := FromASTType(parameter.Type)
			params[i] = Parameter{
				Type: *paramType,
				Name: parameter.Name,
			}
		}

		returnType := FromASTType(t.ReturnType)

		fun := Function{
			Signature: Signature{
				Parameters: params,
				ReturnType: *returnType,
			},
			Body: ToCommand(t.Statement),
		}

		functionType := FunctionType(nil, fun)

		return &LiteralCommand{value: Value{
			Type:  functionType,
			Value: fun,
		}}

	case parser.ContextExpr:
		contextCmd := ExpressionToCommand(t.Context)
		varName := t.Variable.Identifier
		return &ContextCommand{
			contextCmd,
			varName,
		}

	case parser.AssignmentExpr:
		valueCmd := ExpressionToCommand(t.Value)
		//TODO contexts
		name := t.Identifier
		return &AssignmentCommand{
			Name:  name,
			value: valueCmd,
		}
	}

	panic("Could not handle " + reflect.TypeOf(expr).Name())
}
