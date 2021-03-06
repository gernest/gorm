//Package hooks contains callbacks/hooks used by ngorm.
package hooks

import (
	"bytes"
	"errors"
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/gernest/ngorm/builder"
	"github.com/gernest/ngorm/engine"
	"github.com/gernest/ngorm/errmsg"
	"github.com/gernest/ngorm/model"
	"github.com/gernest/ngorm/scope"
	"github.com/gernest/ngorm/search"
	"github.com/gernest/ngorm/util"
)

//Query executes sql QUery without transaction.
func Query(b *Book, e *engine.Engine) error {
	sql, ok := b.Query.Get(model.HookQuerySQL)
	if !ok {
		return errors.New("missing query sql hook")
	}
	err := sql.Exec(b, e)
	if err != nil {
		return err
	}
	exec, ok := b.Query.Get(model.HookQueryExec)
	if !ok {
		return errors.New("missing query exec hook")
	}
	return exec.Exec(b, e)
}

//QueryExec  executes SQL querries.
func QueryExec(b *Book, e *engine.Engine) error {
	var isSlice, isPtr bool
	var resultType reflect.Type
	results := reflect.ValueOf(e.Scope.Value)
	if results.Kind() == reflect.Ptr {
		results = results.Elem()
	}
	if value, ok := e.Scope.Get(model.QueryDestination); ok {
		results = reflect.Indirect(reflect.ValueOf(value))
	}
	if kind := results.Kind(); kind == reflect.Slice {
		isSlice = true
		resultType = results.Type().Elem()
		results.Set(reflect.MakeSlice(results.Type(), 0, 0))

		if resultType.Kind() == reflect.Ptr {
			isPtr = true
			resultType = resultType.Elem()
		}
	} else if kind != reflect.Struct {
		return errors.New("unsupported destination, should be slice or struct")
	}
	e.RowsAffected = 0
	if str, ok := e.Scope.Get(model.QueryOption); ok {
		e.Scope.SQL += util.AddExtraSpaceIfExist(fmt.Sprint(str))
	}

	rows, err := e.SQLDB.Query(e.Scope.SQL, e.Scope.SQLVars...)
	if err != nil {
		return err
	}
	defer func() {
		_ = rows.Close()
	}()

	columns, _ := rows.Columns()
	for rows.Next() {
		e.RowsAffected++
		elem := results
		if isSlice {
			elem = reflect.New(resultType).Elem()
		}
		fields, err := scope.Fields(e, elem.Addr().Interface())
		if err != nil {
			return err
		}
		scope.Scan(rows, columns, fields)
		if isSlice {
			if isPtr {
				results.Set(reflect.Append(results, elem.Addr()))
			} else {
				results.Set(reflect.Append(results, elem))
			}
		}
	}
	if e.RowsAffected == 0 && !isSlice {
		return errmsg.ErrRecordNotFound
	}
	return nil
}

//QuerySQL generates SQL for queries
func QuerySQL(b *Book, e *engine.Engine) error {
	if orderBy, ok := e.Scope.Get(model.OrderByPK); ok {
		pf, err := scope.PrimaryField(e, e.Scope.Value)
		if err != nil {
		} else {
			search.Order(e, fmt.Sprintf("%v%v %v",
				e.Dialect.QueryFieldName(
					scope.QuotedTableName(e, e.Scope.Value)), scope.Quote(e, pf.DBName), orderBy))
		}

	}
	return builder.PrepareQuery(e, e.Scope.Value)
}

//AfterQuery executes any call back after the  Qeery hook has been executed. Any
//callback registered with qeky model.HookQueryAfterFind will be executed.
func AfterQuery(b *Book, e *engine.Engine) error {
	af, ok := b.Query.Get(model.HookAfterFindQuery)
	if ok {
		return af.Exec(b, e)
	}
	return nil
}

//BeforeCreate a callback executed before crating anew record.
func BeforeCreate(b *Book, e *engine.Engine) error {
	bs, ok := b.Create.Get(model.HookBeforeSave)
	if ok {
		err := bs.Exec(b, e)
		if err != nil {
			return err
		}
	}
	bc, ok := b.Create.Get(model.HookBeforeCreate)
	if ok {
		err := bc.Exec(b, e)
		if err != nil {
			return err
		}
	}
	return nil
}

//Create the hook executed to create a new record.
func Create(b *Book, e *engine.Engine) error {
	var (
		columns, placeholders []string

		// The blank columns with default values
		cv []string
	)
	fds, err := scope.Fields(e, e.Scope.Value)
	if err != nil {
		return err
	}

	for _, field := range fds {
		if scope.ChangeableField(e, field) {
			if field.IsNormal {
				if field.IsBlank && field.HasDefaultValue {
					cv = append(cv, scope.Quote(e, field.DBName))
					e.Scope.Set(model.BlankColWithValue, cv)
				} else if !field.IsPrimaryKey || !field.IsBlank {
					columns = append(columns, scope.Quote(e, field.DBName))
					placeholders = append(placeholders, scope.AddToVars(e, field.Field.Interface()))
				}
			} else if field.Relationship != nil && field.Relationship.Kind == "belongs_to" {
				for _, foreignKey := range field.Relationship.ForeignDBNames {
					foreignField, err := scope.FieldByName(e, e.Scope.Value, foreignKey)
					if err != nil {
						return err
					}
					if !scope.ChangeableField(e, foreignField) {
						columns = append(columns, scope.Quote(e, foreignField.DBName))
						placeholders = append(placeholders, scope.AddToVars(e, foreignField.Field.Interface()))
					}
				}
			}
		}
	}

	var (
		returningColumn = "*"
		tableName       = scope.QuotedTableName(e, e.Scope.Value)

		extraOption string
	)

	primaryField, err := scope.PrimaryField(e, e.Scope.Value)
	if err != nil {
		return err
	}
	if str, ok := e.Scope.Get(model.InsertOptions); ok {
		extraOption = fmt.Sprint(str)
	}

	if primaryField != nil {
		returningColumn = scope.Quote(e, primaryField.DBName)
	}

	lastInsertIDReturningSuffix :=
		e.Dialect.LastInsertIDReturningSuffix(tableName, returningColumn)

	if len(columns) == 0 {
		sql := fmt.Sprintf(
			"INSERT INTO %v DEFAULT VALUES%v%v",
			tableName,
			util.AddExtraSpaceIfExist(extraOption),
			util.AddExtraSpaceIfExist(lastInsertIDReturningSuffix),
		)
		e.Scope.SQL = strings.Replace(sql, "$$", "?", -1)
	} else {
		sql := fmt.Sprintf(
			"INSERT INTO %v (%v) VALUES (%v)%v%v",
			scope.QuotedTableName(e, e.Scope.Value),
			strings.Join(columns, ","),
			strings.Join(placeholders, ","),
			util.AddExtraSpaceIfExist(extraOption),
			util.AddExtraSpaceIfExist(lastInsertIDReturningSuffix),
		)
		e.Scope.SQL = strings.Replace(sql, "$$", "?", -1)
	}

	return nil
}

//CreateExec executes the INSERT query and assigns primary key if it is not set
//assuming the primary key is the ID field.
func CreateExec(b *Book, e *engine.Engine) error {
	primaryField, err := scope.PrimaryField(e, e.Scope.Value)
	if err != nil {
		return err
	}
	returningColumn := "*"
	if primaryField != nil {
		returningColumn = scope.Quote(e, primaryField.DBName)
	}
	tableName := scope.QuotedTableName(e, e.Scope.Value)
	lastInsertIDReturningSuffix :=
		e.Dialect.LastInsertIDReturningSuffix(tableName, returningColumn)
	if lastInsertIDReturningSuffix == "" || primaryField == nil {
		tx, err := e.SQLDB.Begin()
		if err != nil {
			return err
		}
		result, err := tx.Exec(e.Scope.SQL, e.Scope.SQLVars...)
		if err != nil {
			rerr := tx.Rollback()
			if rerr != nil {
				return rerr
			}
			return err
		}
		err = tx.Commit()
		if err != nil {
			return err
		}
		// set rows affected count
		e.RowsAffected, _ = result.RowsAffected()

		// set primary value to primary field
		if primaryField != nil && primaryField.IsBlank {
			primaryValue, err := result.LastInsertId()
			if err != nil {
				return err
			}
			_ = primaryField.Set(primaryValue)
		}
	} else {
		if primaryField.Field.CanAddr() {
			err := e.SQLDB.QueryRow(
				e.Scope.SQL,
				e.Scope.SQLVars...,
			).Scan(primaryField.Field.Addr().Interface())
			if err != nil {
				return err
			}
			primaryField.IsBlank = false
			e.RowsAffected = 1
		} else {
			return errmsg.ErrUnaddressable
		}
	}
	return nil
}

//QLAfterCreate hook executed after a new record has been created. This is for
//ql dialect use only.
func QLAfterCreate(b *Book, e *engine.Engine) error {
	ne := cloneEngine(e)
	ne.Scope.Set(model.IgnoreProtectedAttrs, true)
	ne.Scope.Set(model.UpdateInterface, util.ToSearchableMap(e.Scope.Value))
	ne.Scope.Value = e.Scope.Value
	u, ok := b.Update.Get(model.HookUpdateSQL)
	err := u.Exec(b, ne)
	if err != nil {
		return err
	}
	if !ok {
		return errors.New("missing update sql hook")
	}
	err = fixWhere(ne.Scope)
	if err != nil {
		return err
	}
	exec, ok := b.Update.Get(model.HookUpdateExec)
	if !ok {
		return errors.New("missing update exec hook")
	}
	return exec.Exec(b, ne)
}

func fixWhere(s *model.Scope) error {
	src := s.SQL
	i := " id = "
	rep := " id()= "
	w := "WHERE"
	lastWhere := strings.LastIndex(src, w)
	if lastWhere == -1 {
		return nil
	}
	lastID := strings.LastIndex(src, i)
	if lastID == -1 {
		return nil
	}
	if lastID < lastWhere {
		return nil
	}
	s.SQL = src[:lastID] + rep + src[lastID+len(i):]
	n := lastID + len(i) + 1
	ni, err := strconv.Atoi(string(src[n]))
	if err != nil {
		return err
	}
	ni--
	nv := s.SQLVars[ni]
	switch v := nv.(type) {
	case uint64:
		s.SQLVars[ni] = int64(v)
	}
	return nil
}

//BeforeUpdate handles preparations for updating records. This just calls two
//hooks.
//
//	model.HookBeforeSave
//
// If this hook succeeds then It calls
//
//	model.HookBeforeUpdate
func BeforeUpdate(b *Book, e *engine.Engine) error {
	if !scope.HasConditions(e, e.Scope.Value) {
		return errors.New("missing WHERE condition for update")
	}
	if _, ok := e.Scope.Get(model.UpdateColumn); !ok {
		if bs, ok := b.Save.Get(model.HookBeforeSave); ok {
			err := bs.Exec(b, e)
			if err != nil {
				return err
			}
		}
		if bu, ok := b.Update.Get(model.HookBeforeUpdate); ok {
			err := bu.Exec(b, e)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

//AfterUpdate handles things needed to be done after updating records. This just
//calls two hooks
//
//	model.HookAfterUpdate
//
// If this hook succeeds then It calls
//
//	model.HookAfterSave
func AfterUpdate(b *Book, e *engine.Engine) error {
	if !scope.HasConditions(e, e.Scope.Value) {
		return errors.New("missing WHERE condition for update")
	}
	if _, ok := e.Scope.Get(model.UpdateColumn); !ok {
		if au, ok := b.Update.Get(model.HookAfterUpdate); ok {
			err := au.Exec(b, e)
			if err != nil {
				return err
			}
		}
		if as, ok := b.Save.Get(model.HookAfterSave); ok {
			err := as.Exec(b, e)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

//UpdateTimestamp sets the value of UpdatedAt field.
func UpdateTimestamp(b *Book, e *engine.Engine) error {
	if _, ok := e.Scope.Get(model.UpdateColumn); !ok {
		return scope.SetColumn(e, "UpdatedAt", time.Now())
	}
	return nil
}

//AssignUpdatingAttrs assigns value for the attributes that are supposed to be
//updated.
func AssignUpdatingAttrs(b *Book, e *engine.Engine) error {
	if attrs, ok := e.Scope.Get(model.UpdateInterface); ok {
		if u, uok := scope.UpdatedAttrsWithValues(e, attrs); uok {
			e.Scope.Set(model.UpdateAttrs, u)
		}
	}
	return nil
}

//SaveBeforeAssociation saves associations on the model
func SaveBeforeAssociation(b *Book, e *engine.Engine) error {
	if !scope.ShouldSaveAssociation(e) {
		return nil
	}
	fds, err := scope.Fields(e, e.Scope.Value)
	if err != nil {
		return err
	}
	for _, field := range fds {
		if ok, relationship := scope.SaveFieldAsAssociation(e, field); ok && relationship.Kind == "belongs_to" {
			fieldValue := field.Field.Addr().Interface()

			// For the fieldValue, we need to make sure the value is saved into
			// the database.
			//
			// We have two hooks to use here, one model.HookCreateSQL which will
			// build sql for creating the new record and model.HookCreateExec
			// which will execute the generates SQL.
			c, ok := b.Create.Get(model.HookCreateSQL)
			if !ok {
				return errors.New("missing create sql hook")
			}
			ne := cloneEngine(e)
			ne.Scope.Value = fieldValue
			err = c.Exec(b, ne)
			if err != nil {
				return err
			}
			ce, ok := b.Create.Get(model.HookCreateExec)
			if !ok {
				return errors.New("missing create exec hook")
			}
			err = ce.Exec(b, ne)
			if err != nil {
				return err
			}
			if len(relationship.ForeignFieldNames) != 0 {
				// set value's foreign key
				for idx, fieldName := range relationship.ForeignFieldNames {
					associationForeignName := relationship.AssociationForeignDBNames[idx]
					foreignField, err := scope.FieldByName(e, fieldValue, associationForeignName)
					if err != nil {
						//TODO log this?
					} else {
						err = scope.SetColumn(e, fieldName, foreignField.Field.Interface())
						if err != nil {
							return err
						}
					}
				}
			}
		}
	}
	return nil
}

//CreateSQL generates SQL for creating new record
func CreateSQL(b *Book, e *engine.Engine) error {
	if bc, ok := b.Create.Get(model.BeforeCreate); ok {
		err := bc.Exec(b, e)
		if err != nil {
			return err
		}
	}

	if scope.ShouldSaveAssociation(e) {
		if ba, ok := b.Create.Get(model.HookSaveBeforeAss); ok {
			err := ba.Exec(b, e)
			if err != nil {
				return err
			}
		}
	}
	if ts, ok := b.Create.Get(model.HookUpdateTimestamp); ok {
		err := ts.Exec(b, e)
		if err != nil {
			return err
		}
	}
	if c, ok := b.Create.Get(model.Create); ok {
		err := c.Exec(b, e)
		if err != nil {
			return err
		}
	}
	var buf bytes.Buffer
	_, _ = buf.WriteString("BEGIN TRANSACTION;\n")
	if e.Scope.MultiExpr {
		for _, expr := range e.Scope.Exprs {
			_, _ = buf.WriteString("\t" + expr.Q + ";\n")
		}
	}
	_, _ = buf.WriteString("\t" + e.Scope.SQL + ";\n")
	_, _ = buf.WriteString("COMMIT;")
	e.Scope.SQL = buf.String()
	return nil
}

func cloneEngine(e *engine.Engine) *engine.Engine {
	return &engine.Engine{
		Scope:         model.NewScope(),
		Search:        &model.Search{},
		SingularTable: e.SingularTable,
		Ctx:           e.Ctx,
		Dialect:       e.Dialect,
		StructMap:     e.StructMap,
		SQLDB:         e.SQLDB,
		Log:           e.Log,
	}
}

//UpdateSQL builds query for updating records.
func UpdateSQL(b *Book, e *engine.Engine) error {
	var sqls []string
	if up, ok := b.Update.Get(model.HookAssignUpdatingAttrs); ok {
		err := up.Exec(b, e)
		if err != nil {
			return err
		}
	}

	if updateAttrs, ok := e.Scope.Get(model.UpdateAttrs); ok {
		for column, value := range updateAttrs.(map[string]interface{}) {
			sqls = append(sqls, fmt.Sprintf("%v = %v",
				scope.Quote(e, column),
				scope.AddToVars(e, value)))
		}
	} else {
		fds, err := scope.Fields(e, e.Scope.Value)
		if err != nil {
			return err
		}
		for _, field := range fds {
			if scope.ChangeableField(e, field) {
				if !field.IsPrimaryKey && field.IsNormal {
					sqls = append(sqls, fmt.Sprintf("%v = %v",
						scope.Quote(e, field.DBName),
						scope.AddToVars(e, field.Field.Interface())))
				} else if rel := field.Relationship; rel != nil && rel.Kind == "belongs_to" {
					for _, foreignKey := range rel.ForeignDBNames {
						foreignField, err := scope.FieldByName(e, e.Scope.Value, foreignKey)
						if err != nil {
							//TODO log this?
						} else {
							if !scope.ChangeableField(e, foreignField) {
								sqls = append(sqls,
									fmt.Sprintf("%v = %v",
										scope.Quote(e, foreignField.DBName),
										scope.AddToVars(e, foreignField.Field.Interface())))
							}
						}
					}
				}
			}
		}
	}

	var extraOption string
	if str, ok := e.Scope.Get(model.UpdateOptions); ok {
		extraOption = fmt.Sprint(str)
	}

	if len(sqls) > 0 {
		c, err := builder.CombinedCondition(e, e.Scope.Value)
		if err != nil {
			return err
		}
		e.Scope.SQL = fmt.Sprintf(
			"UPDATE %v SET %v%v%v",
			scope.QuotedTableName(e, e.Scope.Value),
			strings.Join(sqls, ", "),
			util.AddExtraSpaceIfExist(c),
			util.AddExtraSpaceIfExist(extraOption),
		)

	}
	var buf bytes.Buffer
	_, _ = buf.WriteString("BEGIN TRANSACTION;\n")
	_, _ = buf.WriteString("\t" + e.Scope.SQL + ";\n")
	_, _ = buf.WriteString("COMMIT;")
	e.Scope.SQL = buf.String()
	return nil
}

//UpdateExec executes UPDATE sql. This assumes the query is already in
//e.Scope.SQL.
func UpdateExec(b *Book, e *engine.Engine) error {
	if e.Scope.SQL == "" {
		return errors.New("missing update sql ")
	}
	tx, err := e.SQLDB.Begin()
	if err != nil {
		return err
	}
	result, err := tx.Exec(e.Scope.SQL, e.Scope.SQLVars...)
	if err != nil {
		rerr := tx.Rollback()
		if rerr != nil {
			return rerr
		}
		return err
	}
	r, err := result.RowsAffected()
	if err != nil {
		return err
	}
	e.RowsAffected = r
	return tx.Commit()
}

//Update generates and executes sql query for updating records.This reliesn on
//two hooks.
//	model.HookUpdateSQL
// Which generates the sql for UPDATE
//
//	model.HookUpdateExec
//which executes the UPDATE sql.
func Update(b *Book, e *engine.Engine) error {
	sql, ok := b.Update.Get(model.HookUpdateSQL)
	if !ok {
		return errors.New("missing update sql hook")
	}
	err := sql.Exec(b, e)
	if err != nil {
		return err
	}
	exec, ok := b.Update.Get(model.HookUpdateExec)
	if !ok {
		return errors.New("missing update exec hook")
	}
	return exec.Exec(b, e)
}

func DeleteSQL(b *Book, e *engine.Engine) error {
	var extraOption string
	if str, ok := e.Scope.Get(model.DeleteOption); ok {
		extraOption = fmt.Sprint(str)
	}

	if e.Dialect.HasColumn(scope.TableName(e, e.Scope.Value), "DeletedAt") {
		c, err := builder.CombinedCondition(e, e.Scope.Value)
		if err != nil {
			return err
		}
		e.Scope.SQL = util.WrapTX(fmt.Sprintf(
			"UPDATE %v SET deleted_at=%v%v%v",
			scope.QuotedTableName(e, e.Scope.Value),
			scope.AddToVars(e, e.Now()),
			util.AddExtraSpaceIfExist(c),
			util.AddExtraSpaceIfExist(extraOption),
		))
	} else {
		c, err := builder.CombinedCondition(e, e.Scope.Value)
		if err != nil {
			return err
		}
		e.Scope.SQL = util.WrapTX(fmt.Sprintf(
			"DELETE FROM %v%v%v",
			scope.QuotedTableName(e, e.Scope.Value),
			util.AddExtraSpaceIfExist(c),
			util.AddExtraSpaceIfExist(extraOption),
		))
	}
	return nil
}

func BeforeDelete(b *Book, e *engine.Engine) error {
	if !scope.HasConditions(e, e.Scope.Value) {
		return errors.New("Missing WHERE clause while deleting")
	}
	if bd, ok := b.Delete.Get(model.HookBeforeDelete); ok {
		return bd.Exec(b, e)
	}
	return nil
}

func AfterDelete(b *Book, e *engine.Engine) error {
	if ad, ok := b.Delete.Get(model.HookAfterDelete); ok {
		return ad.Exec(b, e)
	}
	return nil
}

func Delete(b *Book, e *engine.Engine) error {
	bd, ok := b.Delete.Get(model.BeforeDelete)
	if !ok {
		return errors.New("missing before delete hook")
	}
	err := bd.Exec(b, e)
	if err != nil {
		return err
	}
	sql, ok := b.Delete.Get(model.DeleteSQL)
	if !ok {
		return errors.New("missing before delete hook")
	}
	err = sql.Exec(b, e)
	if err != nil {
		return err
	}
	tx, err := e.SQLDB.Begin()
	if err != nil {
		return err
	}
	result, err := tx.Exec(e.Scope.SQL, e.Scope.SQLVars...)
	if err != nil {
		_ = tx.Rollback()
		return err
	}
	a, err := result.RowsAffected()
	if err != nil {
		return err
	}
	e.RowsAffected = a
	err = tx.Commit()
	if err != nil {
		return err
	}
	ad, ok := b.Delete.Get(model.AfterDelete)
	if !ok {
		return errors.New("missing after delete hook")
	}
	return ad.Exec(b, e)
}
