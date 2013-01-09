package hood

import (
	"database/sql"
	"os"
	"testing"
	"time"
)

// THE IMPORTS AND LIVE TESTS ARE DISABLED BY DEFAULT, NOT TO INTERFERE WITH
// REAL UNIT TESTS, SINCE THEY DO REQUIRE A CERTAIN SYSTEM CONFIGURATION!
//
// ONLY ENABLE THE LIVE TESTS IF NECESSARY
//
// TO ENABLE/DISABLE LIVE TESTS UNCOMMENT/COMMENT THE DRIVER IMPORT AND THE
// CORRESPONDING DIALECT INFO IN THE TO_RUN ARRAY!

import (
// _ "github.com/bmizerany/pq"
// _ "github.com/mattn/go-sqlite3"
)

var toRun = []dialectInfo{
// dialectInfo{
// 	NewPostgres(),
// 	setupPgDb,
// 	`CREATE TABLE without_pk ( first text, last text, amount integer )`,
// 	`CREATE TABLE IF NOT EXISTS without_pk ( first text, last text, amount integer )`,
// 	`CREATE TABLE with_pk ( primary bigserial PRIMARY KEY, first text, last text, amount integer )`,
// 	`INSERT INTO sql_gen_model (first, last, amount) VALUES ($1, $2, $3) RETURNING prim`,
// 	`UPDATE sql_gen_model SET first = $1, last = $2, amount = $3 WHERE prim = $4`,
// 	`DELETE FROM sql_gen_model WHERE prim = $1`,
// 	`SELECT * FROM sql_gen_model INNER JOIN orders ON users.id == orders.id WHERE id = $1 AND category_id = $2 GROUP BY name HAVING SUM(price) < $3 ORDER BY first_name LIMIT $4 OFFSET $5`,
// 	`DROP TABLE drop_table`,
// 	`DROP TABLE IF EXISTS drop_table`,
// 	`ALTER TABLE table_a RENAME TO table_b`,
// 	`ALTER TABLE a ADD COLUMN c varchar(100)`,
// 	`ALTER TABLE a RENAME COLUMN b TO c`,
// 	`ALTER TABLE a ALTER COLUMN b TYPE varchar(100)`,
// 	`ALTER TABLE a DROP COLUMN b`,
// 	`CREATE UNIQUE INDEX iname ON itable (a, b, c)`,
// 	`CREATE INDEX iname2 ON itable2 (d, e)`,
// 	`DROP INDEX iname`,
// },
// dialectInfo{
// 	NewSqlite3(),
// 	setupSqlite3Db,
// 	`CREATE TABLE without_pk ( first text, last text, amount integer )`,
// 	`CREATE TABLE IF NOT EXISTS without_pk ( first text, last text, amount integer )`,
// 	`CREATE TABLE with_pk ( primary integer PRIMARY KEY AUTOINCREMENT, first text, last text, amount integer )`,
// 	`INSERT INTO sql_gen_model (first, last, amount) VALUES ($1, $2, $3)`,
// 	`UPDATE sql_gen_model SET first = $1, last = $2, amount = $3 WHERE prim = $4`,
// 	`DELETE FROM sql_gen_model WHERE prim = $1`,
// 	`SELECT * FROM sql_gen_model INNER JOIN orders ON users.id == orders.id WHERE id = $1 AND category_id = $2 GROUP BY name HAVING SUM(price) < $3 ORDER BY first_name LIMIT $4 OFFSET $5`,
// 	`DROP TABLE drop_table`,
// 	`DROP TABLE IF EXISTS drop_table`,
// 	`ALTER TABLE table_a RENAME TO table_b`,
// 	`ALTER TABLE a ADD COLUMN c text`,
// 	``, // not supported by sql command
// 	``, // not supported by sql command
// 	``, // not supported by sql command
// 	`CREATE UNIQUE INDEX iname ON itable (a, b, c)`,
// 	`CREATE INDEX iname2 ON itable2 (d, e)`,
// 	`DROP INDEX iname`,
// },
}

type dialectInfo struct {
	dialect                         Dialect
	setupDbFunc                     func(t *testing.T) *Hood
	createTableWithoutPkSql         string
	createTableWithoutPkIfExistsSql string
	createTableWithPkSql            string
	insertSql                       string
	updateSql                       string
	deleteSql                       string
	querySql                        string
	dropTableSql                    string
	dropTableIfExistsSql            string
	renameTableSql                  string
	addColumnSql                    string
	renameColumnSql                 string
	changeColumnSql                 string
	dropColumnSql                   string
	createUniqueIndexSql            string
	createIndexSql                  string
	dropIndexSql                    string
}

func setupPgDb(t *testing.T) *Hood {
	db, err := sql.Open("postgres", "user=hood dbname=hood_test sslmode=disable")
	if err != nil {
		t.Fatal("could not open db", err)
	}
	hd := New(db, NewPostgres())
	hd.Log = true
	return hd
}

func setupSqlite3Db(t *testing.T) *Hood {
	os.Remove("/tmp/foo.db")
	db, err := sql.Open("sqlite3", "/tmp/foo.db")
	if err != nil {
		t.Fatal("could not open db", err)
	}
	hd := New(db, NewSqlite3())
	hd.Log = true
	return hd
}

func TestTransaction(t *testing.T) {
	for _, info := range toRun {
		DoTestTransaction(t, info)
	}
}

func DoTestTransaction(t *testing.T, info dialectInfo) {
	hd := info.setupDbFunc(t)
	type txModel struct {
		Id Id
		A  string
	}
	table := txModel{
		A: "A",
	}

	hd.DropTable(&table)
	err := hd.CreateTable(&table)
	if err != nil {
		t.Fatal("error not nil", err)
	}

	tx := hd.Begin()
	if _, ok := hd.qo.(*sql.DB); !ok {
		t.Fatal("wrong type")
	}
	if _, ok := tx.qo.(*sql.Tx); !ok {
		t.Fatal("wrong type")
	}
	_, err = tx.Save(&table)
	if err != nil {
		t.Fatal("error not nil", err)
	}
	err = tx.Rollback()
	if err != nil {
		t.Fatal("error not nil", err)
	}

	var out []txModel
	err = hd.Find(&out)
	if err != nil {
		t.Fatal("error not nil", err)
	}
	if x := len(out); x > 0 {
		t.Fatal("wrong length", x)
	}

	tx = hd.Begin()
	table.Id = 0 // force insert by resetting id
	_, err = tx.Save(&table)
	if err != nil {
		t.Fatal("error not nil", err)
	}
	err = tx.Commit()
	if err != nil {
		t.Fatal("error not nil", err)
	}

	out = nil
	err = hd.Find(&out)
	if err != nil {
		t.Fatal("error not nil", err)
	}
	if x := len(out); x != 1 {
		t.Fatal("wrong length", x)
	}
}

func TestSaveAndDelete(t *testing.T) {
	for _, info := range toRun {
		DoTestSaveAndDelete(t, info)
	}
}

func DoTestSaveAndDelete(t *testing.T, info dialectInfo) {
	hd := info.setupDbFunc(t)
	type saveModel struct {
		Id Id
		A  string
		B  int
	}
	model1 := saveModel{
		A: "banana",
		B: 5,
	}
	model2 := saveModel{
		A: "orange",
		B: 4,
	}

	hd.DropTable(&model1)

	err := hd.CreateTable(&model1)
	if err != nil {
		t.Fatal("error not nil", err)
	}
	id, err := hd.Save(&model1)
	if err != nil {
		t.Fatal("error not nil", err)
	}
	if id != 1 {
		t.Fatal("wrong id", id)
	}

	model1.A = "grape"
	model1.B = 9

	id, err = hd.Save(&model1)
	if err != nil {
		t.Fatal("error not nil", err)
	}
	if id != 1 {
		t.Fatal("wrong id", id)
	}

	id, err = hd.Save(&model2)
	if err != nil {
		t.Fatal("error not nil", err)
	}
	if id != 2 {
		t.Fatal("wrong id", id)
	}
	if model2.Id != id {
		t.Fatal("id should have been copied", model2.Id)
	}

	id2, err := hd.Delete(&model2)
	if err != nil {
		t.Fatal("error not nil", err)
	}
	if id != id2 {
		t.Fatal("wrong id", id, id2)
	}
}

func TestSaveDeleteAllAndHooks(t *testing.T) {
	for _, info := range toRun {
		DoTestSaveDeleteAllAndHooks(t, info)
	}
}

type sdAllModel struct {
	Id Id
	A  string
}

var sdAllHooks []string

func (m *sdAllModel) BeforeSave() error {
	sdAllHooks = append(sdAllHooks, "bsave")
	return nil
}

func (m *sdAllModel) AfterSave() error {
	sdAllHooks = append(sdAllHooks, "asave")
	return nil
}

func (m *sdAllModel) BeforeInsert() error {
	sdAllHooks = append(sdAllHooks, "binsert")
	return nil
}

func (m *sdAllModel) AfterInsert() error {
	sdAllHooks = append(sdAllHooks, "ainsert")
	return nil
}

func (m *sdAllModel) BeforeUpdate() error {
	sdAllHooks = append(sdAllHooks, "bupdate")
	return nil
}

func (m *sdAllModel) AfterUpdate() error {
	sdAllHooks = append(sdAllHooks, "aupdate")
	return nil
}

func (m *sdAllModel) BeforeDelete() error {
	sdAllHooks = append(sdAllHooks, "bdelete")
	return nil
}

func (m *sdAllModel) AfterDelete() error {
	sdAllHooks = append(sdAllHooks, "adelete")
	return nil
}

func DoTestSaveDeleteAllAndHooks(t *testing.T, info dialectInfo) {
	hd := info.setupDbFunc(t)
	hd.DropTable(&sdAllModel{})

	models := []sdAllModel{
		sdAllModel{A: "A"},
		sdAllModel{A: "B"},
	}

	sdAllHooks = make([]string, 0, 20)
	err := hd.CreateTable(&sdAllModel{})
	if err != nil {
		t.Fatal("error not nil", err)
	}

	ids, err := hd.SaveAll(&models)
	if err != nil {
		t.Fatal("error not nil", err)
	}
	if x := len(ids); x != 2 {
		t.Fatal("wrong id count", x)
	}
	if x := ids[0]; x != 1 {
		t.Fatal("wrong id", x)
	}
	if x := ids[1]; x != 2 {
		t.Fatal("wrong id", x)
	}
	if x := models[0].Id; x != 1 {
		t.Fatal("wrong id", x)
	}
	if x := models[1].Id; x != 2 {
		t.Fatal("wrong id", x)
	}

	hd.SaveAll(&models) // force update for hooks test 

	_, err = hd.DeleteAll(&models)
	if err != nil {
		t.Fatal("error not nil", err)
	}

	if x := len(sdAllHooks); x != 20 {
		t.Fatal("wrong hook call count", x)
	}
	hookMatch := []string{
		"bsave",
		"binsert",
		"ainsert",
		"asave",
		"bsave",
		"binsert",
		"ainsert",
		"asave",
		"bsave",
		"bupdate",
		"aupdate",
		"asave",
		"bsave",
		"bupdate",
		"aupdate",
		"asave",
		"bdelete",
		"adelete",
		"bdelete",
		"adelete",
	}
	for i, v := range hookMatch {
		if x := sdAllHooks[i]; x != v {
			t.Fatal("wrong hook sequence", x, v)
		}
	}
}

func TestFind(t *testing.T) {
	for _, info := range toRun {
		DoTestFind(t, info)
	}
}

func DoTestFind(t *testing.T, info dialectInfo) {
	hd := info.setupDbFunc(t)
	now := time.Now()

	type findModel struct {
		Id Id
		A  string
		B  int
		C  int8
		D  int16
		E  int32
		F  int64
		G  uint
		H  uint8
		I  uint16
		J  uint32
		K  uint64
		L  float32
		M  float64
		N  []byte
		O  VarChar
		P  time.Time
	}
	model1 := findModel{
		A: "string!",
		B: -1,
		C: -2,
		D: -3,
		E: -4,
		F: -5,
		G: 6,
		H: 7,
		I: 8,
		J: 9,
		K: 10,
		L: 11.5,
		M: 12.6,
		N: []byte("bytes!"),
		O: "varchar!",
		P: now,
	}

	hd.DropTable(&model1)

	err := hd.CreateTable(&model1)
	if err != nil {
		t.Fatal("error not nil", err)
	}

	var out []findModel
	err = hd.Where("a = ? AND j = ?", "string!", 9).Find(&out)
	if err != nil {
		t.Fatal("error not nil", err)
	}
	if out != nil {
		t.Fatal("output should be nil", out)
	}

	id, err := hd.Save(&model1)
	if err != nil {
		t.Fatal("error not nil", err)
	}
	if id != 1 {
		t.Fatal("wrong id", id)
	}

	err = hd.Where("a = ? AND j = ?", "string!", 9).Find(&out)
	if err != nil {
		t.Fatal("error not nil", err)
	}
	if out == nil {
		t.Fatal("output should not be nil")
	}
	if x := len(out); x != 1 {
		t.Fatal("invalid output length", x)
	}
	for _, v := range out {
		if x := v.Id; x != 1 {
			t.Fatal("invalid value", x)
		}
		if x := v.A; x != "string!" {
			t.Fatal("invalid value", x)
		}
		if x := v.B; x != -1 {
			t.Fatal("invalid value", x)
		}
		if x := v.C; x != -2 {
			t.Fatal("invalid value", x)
		}
		if x := v.D; x != -3 {
			t.Fatal("invalid value", x)
		}
		if x := v.E; x != -4 {
			t.Fatal("invalid value", x)
		}
		if x := v.F; x != -5 {
			t.Fatal("invalid value", x)
		}
		if x := v.G; x != 6 {
			t.Fatal("invalid value", x)
		}
		if x := v.H; x != 7 {
			t.Fatal("invalid value", x)
		}
		if x := v.I; x != 8 {
			t.Fatal("invalid value", x)
		}
		if x := v.J; x != 9 {
			t.Fatal("invalid value", x)
		}
		if x := v.K; x != 10 {
			t.Fatal("invalid value", x)
		}
		if x := v.L; x != 11.5 {
			t.Fatal("invalid value", x)
		}
		if x := v.M; x != 12.6 {
			t.Fatal("invalid value", x)
		}
		if x := v.N; string(x) != "bytes!" {
			t.Fatal("invalid value", x)
		}
		if x := v.O; string(x) != "varchar!" {
			t.Fatal("invalid value", x)
		}
		if x := v.P; now.Equal(x) {
			t.Fatal("invalid value", x)
		}
	}

	model1.Id = 0 // force insert, would update otherwise
	model1.A = "row2"

	id, err = hd.Save(&model1)
	if err != nil {
		t.Fatal("error not nil", err)
	}
	if id != 2 {
		t.Fatal("wrong id", id)
	}

	out = nil
	err = hd.Where("a = ? AND j = ?", "row2", 9).Find(&out)
	if err != nil {
		t.Fatal("error not nil", err)
	}
	if x := len(out); x != 1 {
		t.Fatal("invalid output length", x)
	}

	out = nil
	err = hd.Where("j = ?", 9).Find(&out)
	if err != nil {
		t.Fatal("error not nil", err)
	}
	if x := len(out); x != 2 {
		t.Fatal("invalid output length", x)
	}
}

func TestCreateTable(t *testing.T) {
	for _, info := range toRun {
		DoTestCreateTable(t, info)
	}
}

func DoTestCreateTable(t *testing.T, info dialectInfo) {
	hd := info.setupDbFunc(t)
	type model struct {
		Prim   Id
		First  string `sql:"notnull"`
		Last   string `sql:"default('defaultValue')"`
		Amount int
	}
	table := &model{}

	hd.DropTable(table)
	err := hd.CreateTable(table)
	if err != nil {
		t.Fatal("error not nil", err)
	}
	err = hd.DropTable(table)
	if err != nil {
		t.Fatal("error not nil", err)
	}
}

func TestCreateTableSql(t *testing.T) {
	for _, info := range toRun {
		DoTestCreateTableSql(t, info)
	}
}

func DoTestCreateTableSql(t *testing.T, info dialectInfo) {
	type withoutPk struct {
		First  string
		Last   string
		Amount int
	}
	table := &withoutPk{"a", "b", 5}
	model, err := interfaceToModel(table)
	if err != nil {
		t.Fatal("error not nil", err)
	}
	if x := info.dialect.CreateTableSql(model, false); x != info.createTableWithoutPkSql {
		t.Fatal("wrong sql", x)
	}
	if x := info.dialect.CreateTableSql(model, true); x != info.createTableWithoutPkIfExistsSql {
		t.Fatal("wrong sql", x)
	}
	type withPk struct {
		Primary Id
		First   string
		Last    string
		Amount  int
	}
	table2 := &withPk{First: "a", Last: "b", Amount: 5}
	model, err = interfaceToModel(table2)
	if err != nil {
		t.Fatal("error not nil", err)
	}
	if x := info.dialect.CreateTableSql(model, false); x != info.createTableWithPkSql {
		t.Fatal("wrong query", x)
	}
}

type sqlGenModel struct {
	Prim   Id
	First  string
	Last   string
	Amount int
}

var sqlGenSampleData = &sqlGenModel{3, "FirstName", "LastName", 6}

func TestInsertSQL(t *testing.T) {
	for _, info := range toRun {
		DoTestInsertSQL(t, info)
	}
}

func DoTestInsertSQL(t *testing.T, info dialectInfo) {
	model, _ := interfaceToModel(sqlGenSampleData)
	sql, _ := info.dialect.InsertSql(model)
	if x := info.insertSql; x != sql {
		t.Fatalf("invalid sql: '%v'", sql, x)
	}
}

func TestUpdateSQL(t *testing.T) {
	for _, info := range toRun {
		DoTestUpdateSQL(t, info)
	}
}

func DoTestUpdateSQL(t *testing.T, info dialectInfo) {
	model, _ := interfaceToModel(sqlGenSampleData)
	sql, _ := info.dialect.UpdateSql(model)
	if x := info.updateSql; x != sql {
		t.Fatalf("invalid sql: '%v'", sql, x)
	}
}

func TestDeleteSQL(t *testing.T) {
	for _, info := range toRun {
		DoTestDeleteSQL(t, info)
	}
}

func DoTestDeleteSQL(t *testing.T, info dialectInfo) {
	model, _ := interfaceToModel(sqlGenSampleData)
	sql, _ := info.dialect.DeleteSql(model)
	if x := info.deleteSql; x != sql {
		t.Fatalf("invalid sql: '%v'", sql, x)
	}
}

func TestQuerySQL(t *testing.T) {
	for _, info := range toRun {
		DoTestQuerySQL(t, info)
	}
}

func DoTestQuerySQL(t *testing.T, info dialectInfo) {
	hood := New(nil, info.dialect)
	hood.Select("*", &sqlGenModel{})
	hood.Where("id = ?", 2)
	hood.Where("category_id = ?", 5)
	hood.Join("INNER", "orders", "users.id == orders.id")
	hood.GroupBy("name")
	hood.Having("SUM(price) < ?", 2000)
	hood.OrderBy("first_name")
	hood.Offset(3)
	hood.Limit(10)
	query, _ := hood.Dialect.QuerySql(hood)
	// TODO: verify 2nd argument ARGS
	if x := info.querySql; x != query {
		t.Fatalf("invalid query: '%v'", query, x)
	}
}

func TestDropTableSQL(t *testing.T) {
	for _, info := range toRun {
		DoTestDropTableSQL(t, info)
	}
}

func DoTestDropTableSQL(t *testing.T, info dialectInfo) {
	if x := info.dialect.DropTableSql("drop_table", false); x != info.dropTableSql {
		t.Fatal("wrong sql", x)
	}
	if x := info.dialect.DropTableSql("drop_table", true); x != info.dropTableIfExistsSql {
		t.Fatal("wrong sql", x)
	}
}

func TestRenameTableSQL(t *testing.T) {
	for _, info := range toRun {
		DoTestRenameTableSQL(t, info)
	}
}

func DoTestRenameTableSQL(t *testing.T, info dialectInfo) {
	if x := info.dialect.RenameTableSql("table_a", "table_b"); x != info.renameTableSql {
		t.Fatal("wrong sql", x)
	}
}

func TestAddColumSQL(t *testing.T) {
	for _, info := range toRun {
		DoTestAddColumSQL(t, info)
	}
}

func DoTestAddColumSQL(t *testing.T, info dialectInfo) {
	if x := info.dialect.AddColumnSql("a", "c", VarChar(""), 100); x != info.addColumnSql {
		t.Fatal("wrong sql", x)
	}
}

func TestRenameColumnSql(t *testing.T) {
	for _, info := range toRun {
		DoTestRenameColumnSql(t, info)
	}
}

func DoTestRenameColumnSql(t *testing.T, info dialectInfo) {
	if x := info.dialect.RenameColumnSql("a", "b", "c"); x != info.renameColumnSql {
		t.Fatal("wrong sql", x)
	}
}

func TestChangeColumnSql(t *testing.T) {
	for _, info := range toRun {
		DoTestChangeColumnSql(t, info)
	}
}

func DoTestChangeColumnSql(t *testing.T, info dialectInfo) {
	if x := info.dialect.ChangeColumnSql("a", "b", VarChar(""), 100); x != info.changeColumnSql {
		t.Fatal("wrong sql", x)
	}
}

func TestRemoveColumnSql(t *testing.T) {
	for _, info := range toRun {
		DoTestRemoveColumnSql(t, info)
	}
}

func DoTestRemoveColumnSql(t *testing.T, info dialectInfo) {
	if x := info.dialect.DropColumnSql("a", "b"); x != info.dropColumnSql {
		t.Fatal("wrong sql", x)
	}
}

func TestCreateIndexSql(t *testing.T) {
	for _, info := range toRun {
		DoTestCreateIndexSql(t, info)
	}
}

func DoTestCreateIndexSql(t *testing.T, info dialectInfo) {
	if x := info.dialect.CreateIndexSql("iname", "itable", true, "a", "b", "c"); x != info.createUniqueIndexSql {
		t.Fatal("wrong sql", x)
	}
	if x := info.dialect.CreateIndexSql("iname2", "itable2", false, "d", "e"); x != info.createIndexSql {
		t.Fatal("wrong sql", x)
	}
}

func TestDropIndexSql(t *testing.T) {
	for _, info := range toRun {
		DoTestDropIndexSql(t, info)
	}
}

func DoTestDropIndexSql(t *testing.T, info dialectInfo) {
	if x := info.dialect.DropIndexSql("iname"); x != info.dropIndexSql {
		t.Fatal("wrong sql", x)
	}
}

func TestSqlTypeForPgDialect(t *testing.T) {
	d := NewPostgres()
	if x := d.SqlType(true, 0); x != "boolean" {
		t.Fatal("wrong type", x)
	}
	var indirect interface{} = true
	if x := d.SqlType(indirect, 0); x != "boolean" {
		t.Fatal("wrong type", x)
	}
	if x := d.SqlType(uint32(2), 0); x != "integer" {
		t.Fatal("wrong type", x)
	}
	if x := d.SqlType(Id(1), 0); x != "bigserial" {
		t.Fatal("wrong type", x)
	}
	if x := d.SqlType(int64(1), 0); x != "bigint" {
		t.Fatal("wrong type", x)
	}
	if x := d.SqlType(1.8, 0); x != "double precision" {
		t.Fatal("wrong type", x)
	}
	if x := d.SqlType([]byte("asdf"), 0); x != "bytea" {
		t.Fatal("wrong type", x)
	}
	if x := d.SqlType("astring", 0); x != "text" {
		t.Fatal("wrong type", x)
	}
	if x := d.SqlType(VarChar("a"), 0); x != "varchar(255)" {
		t.Fatal("wrong type", x)
	}
	if x := d.SqlType(VarChar("b"), 128); x != "varchar(128)" {
		t.Fatal("wrong type", x)
	}
	if x := d.SqlType(time.Now(), 0); x != "timestamp" {
		t.Fatal("wrong type", x)
	}
}

func TestSqlTypeForSqlite3Dialect(t *testing.T) {
	d := NewSqlite3()
	if x := d.SqlType(true, 0); x != "integer" {
		t.Fatal("wrong type", x)
	}
	var indirect interface{} = true
	if x := d.SqlType(indirect, 0); x != "integer" {
		t.Fatal("wrong type", x)
	}
	if x := d.SqlType(uint32(2), 0); x != "integer" {
		t.Fatal("wrong type", x)
	}
	if x := d.SqlType(Id(1), 0); x != "integer" {
		t.Fatal("wrong type", x)
	}
	if x := d.SqlType(int64(1), 0); x != "integer" {
		t.Fatal("wrong type", x)
	}
	if x := d.SqlType(1.8, 0); x != "real" {
		t.Fatal("wrong type", x)
	}
	if x := d.SqlType([]byte("asdf"), 0); x != "text" {
		t.Fatal("wrong type", x)
	}
	if x := d.SqlType("astring", 0); x != "text" {
		t.Fatal("wrong type", x)
	}
	if x := d.SqlType(VarChar("a"), 0); x != "text" {
		t.Fatal("wrong type", x)
	}
	if x := d.SqlType(VarChar("b"), 128); x != "text" {
		t.Fatal("wrong type", x)
	}
	if x := d.SqlType(time.Now(), 0); x != "text" {
		t.Fatal("wrong type", x)
	}
}
