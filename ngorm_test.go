package ngorm

import (
	"strings"
	"testing"
	"time"

	_ "github.com/cznic/ql/driver"
	"github.com/gernest/ngorm/fixture"
)

type Foo struct {
	ID    int
	Stuff string
}

func TestDB_CreateTable(t *testing.T) {
	for _, d := range AllTestDB() {
		runWrapDB(t, d, testDB_CreateTable)
	}
}

func testDB_CreateTable(t *testing.T, db *DB) {
	// create table tests
	sql, err := db.CreateTableSQL(&Foo{})
	if err != nil {
		t.Fatal(err)
	}
	expect := fixture.GetSQL(db.Dialect().GetName(), fixture.CreateTable1)
	expect = strings.TrimSpace(expect)
	if sql.Q != expect {
		t.Errorf("expected %s got %s", expect, sql.Q)
	}
	_, err = db.CreateTable(&Foo{})
	if err != nil {
		t.Error(err)
	}

	// multiple tables
	sql, err = db.CreateTableSQL(
		&fixture.User{},
	)
	if err != nil {
		t.Fatal(err)
	}
	expect = fixture.GetSQL(db.Dialect().GetName(), fixture.CreateTable2)
	if sql.Q != expect {
		t.Errorf("expected %s got %s", expect, sql.Q)
	}
	_, err = db.CreateTable(&fixture.User{})
	if err != nil {
		t.Error(err)
	}
}

func TestDB_DropTable(t *testing.T) {
	for _, d := range AllTestDB() {
		runWrapDB(t, d, testDB_DropTable)
	}
}

func testDB_DropTable(t *testing.T, db *DB) {
	_, err := db.DropTable(&Foo{})
	if err == nil {
		t.Error("expected error")
	}

	_, err = db.CreateTable(&Foo{})
	if err != nil {
		t.Fatal(err)
	}

	_, err = db.DropTable(&Foo{})
	if err != nil {
		t.Fatal(err)
	}

	if db.dialect.HasTable("foos") {
		t.Error("expected the table to disappear")
	}

	sql, err := db.DropTableSQL(&Foo{}, &fixture.User{})
	if err != nil {
		t.Fatal(err)
	}
	expect := fixture.GetSQL(db.Dialect().GetName(), fixture.DropTable)
	if sql.Q != expect {
		t.Errorf("expected %s got %s", expect, sql.Q)
	}
}

func TestDB_Automirate(t *testing.T) {
	for _, d := range AllTestDB() {
		runWrapDB(t, d, testDB_Automirate)
	}
}

func testDB_Automirate(t *testing.T, db *DB) {
	sql, err := db.AutomigrateSQL(
		&fixture.User{},
		&fixture.Email{},
		&fixture.Language{},
		&fixture.Company{},
		&fixture.CreditCard{},
		&fixture.Address{},
	)
	if err != nil {
		t.Fatal(err)
	}

	expect := fixture.GetSQL(db.Dialect().GetName(), fixture.AutoMigrate)
	if sql.Q != expect {
		t.Errorf("expected %s got %s", expect, sql.Q)
	}
	_, err = db.Automigrate(
		&fixture.User{},
		&fixture.Email{},
		&fixture.Language{},
		&fixture.Company{},
		&fixture.CreditCard{},
		&fixture.Address{},
	)
	if err != nil {
		t.Fatal(err)
	}
}

func TestDB_Create(t *testing.T) {
	for _, d := range AllTestDB() {
		runWrapDB(t, d, testDB_Create)
	}
}
func testDB_Create(t *testing.T, db *DB) {
	_, err := db.Automigrate(
		&fixture.User{},
		&fixture.Email{},
		&fixture.Language{},
		&fixture.Company{},
		&fixture.CreditCard{},
		&fixture.Address{},
	)
	if err != nil {
		t.Fatal(err)
	}
	n := time.Now()
	user := fixture.User{Name: "Jinzhu", Age: 18, Birthday: &n}
	_, err = db.CreateSQL(&user)
	if err != nil {
		t.Fatal(err)
	}
	err = db.Create(&user)
	if err != nil {
		t.Fatal(err)
	}
}

func TestDB_SaveSQL(t *testing.T) {
	for _, d := range AllTestDB() {
		runWrapDB(t, d, testDB_SaveSQL)
	}
}

func testDB_SaveSQL(t *testing.T, db *DB) {
	sql, err := db.SaveSQL(&Foo{ID: 10, Stuff: "twenty"})
	if err != nil {
		t.Fatal(err)
	}
	expect := fixture.GetSQL(db.Dialect().GetName(), fixture.SaveSQL)
	if sql.Q != expect {
		t.Errorf("expected %s got %s", expect, sql.Q)
	}
}

func TestDB_UpdateSQL(t *testing.T) {
	for _, d := range AllTestDB() {
		runWrapDB(t, d, testDB_UpdateSQL)
	}
}

func testDB_UpdateSQL(t *testing.T, db *DB) {
	foo := Foo{ID: 10, Stuff: "twenty"}
	sql, err := db.Model(&foo).UpdateSQL("stuff", "hello")
	if err != nil {
		t.Fatal(err)
	}
	expect := fixture.GetSQL(db.Dialect().GetName(), fixture.UpdateSQL)
	if sql.Q != expect {
		t.Errorf("expected %s got %s", expect, sql.Q)
	}
}

func TestDB_SingularTable(t *testing.T) {
	for _, d := range AllTestDB() {
		runWrapDB(t, d, testDB_SingularTable)
	}
}

func testDB_SingularTable(t *testing.T, db *DB) {
	db.SingularTable(true)
	sql, err := db.CreateSQL(&Foo{})
	if err != nil {
		t.Fatal(err)
	}
	expect := fixture.GetSQL(db.Dialect().GetName(), fixture.SingularTable)
	if sql.Q != expect {
		t.Errorf("expected %s got %s", expect, sql.Q)
	}
}

func TestDB_HasTable(t *testing.T) {
	for _, d := range AllTestDB() {
		runWrapDB(t, d, testDB_HasTable)
	}
}

func testDB_HasTable(t *testing.T, db *DB) {
	if db.HasTable("foos") {
		t.Error("expected false")
	}
	if db.HasTable(&Foo{}) {
		t.Error("expected false")
	}
	_, err := db.Automigrate(&Foo{})
	if err != nil {
		t.Fatal(err)
	}
	if !db.HasTable("foos") {
		t.Error("expected true")
	}
	if !db.HasTable(&Foo{}) {
		t.Error("expected true")
	}
}

func TestDB_FirstSQL(t *testing.T) {
	for _, d := range AllTestDB() {
		runWrapDB(t, d, testDB_FirstSQL)
	}
}

func testDB_FirstSQL(t *testing.T, db *DB) {

	// First record order by primary key
	sql, err := db.FirstSQL(&fixture.User{})
	if err != nil {
		t.Fatal(err)
	}
	expect := fixture.GetSQL(db.Dialect().GetName(), fixture.FirstSQL1)
	if sql.Q != expect {
		t.Errorf("expected %s got %s", expect, sql.Q)
	}

	// First record with primary key
	sql, err = db.Begin().FirstSQL(&fixture.User{}, 10)
	if err != nil {
		t.Fatal(err)
	}
	expect = fixture.GetSQL(db.Dialect().GetName(), fixture.FirstSQL2)
	if sql.Q != expect {
		t.Errorf("expected %s got %s", expect, sql.Q)
	}
}

func TestDB_First(t *testing.T) {
	db, err := Open("ql-mem", "test.db")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = db.Close() }()
	_, err = db.Automigrate(&Foo{})
	if err != nil {
		t.Fatal(err)
	}

	sample := []string{"a", "b", "c", "d"}
	for _, v := range sample {
		err := db.Create(&Foo{Stuff: v})
		if err != nil {
			t.Fatal(err)
		}
	}
	fu := Foo{}
	err = db.First(&fu)
	if err != nil {
		t.Fatal(err)
	}
	if fu.Stuff != "a" {
		t.Errorf("expected a got  %s", fu.Stuff)
	}
}

func TestDB_LastSQL(t *testing.T) {
	db, err := Open("ql-mem", "test.db")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = db.Close() }()

	// First record order by primary key
	sql, err := db.LastSQL(&fixture.User{})
	if err != nil {
		t.Fatal(err)
	}
	expect := `SELECT * FROM users   ORDER BY id DESC`
	expect = strings.TrimSpace(expect)
	if sql.Q != expect {
		t.Errorf("expected %s got %s", expect, sql.Q)
	}

	// First record with primary key
	sql, err = db.clone().LastSQL(&fixture.User{}, 10)
	if err != nil {
		t.Fatal(err)
	}
	expect = `SELECT * FROM users  WHERE (id = $1) ORDER BY id DESC`
	expect = strings.TrimSpace(expect)
	if sql.Q != expect {
		t.Errorf("expected %s got %s", expect, sql.Q)
	}
}
func TestDB_Last(t *testing.T) {
	db, err := Open("ql-mem", "test.db")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = db.Close() }()
	_, err = db.Automigrate(&Foo{})
	if err != nil {
		t.Fatal(err)
	}

	sample := []string{"a", "b", "c", "d"}
	for _, v := range sample {
		err := db.Create(&Foo{Stuff: v})
		if err != nil {
			t.Fatal(err)
		}
	}
	fu := Foo{}
	err = db.Last(&fu)
	if err != nil {
		t.Fatal(err)
	}
	if fu.Stuff != "d" {
		t.Errorf("expected d got  %s", fu.Stuff)
	}
}

func TestDB_FindSQL(t *testing.T) {
	db, err := Open("ql-mem", "test.db")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = db.Close() }()

	// First record order by primary key
	users := []*fixture.User{}
	sql, err := db.FindSQL(&users)
	if err != nil {
		t.Fatal(err)
	}
	expect := `SELECT * FROM users`
	expect = strings.TrimSpace(expect)
	sql.Q = strings.TrimSpace(sql.Q)
	if sql.Q != expect {
		t.Errorf("expected %s got %s", expect, sql.Q)
	}

	sql, err = db.clone().Limit(2).FindSQL(&users)
	if err != nil {
		t.Fatal(err)
	}
	expect = `SELECT * FROM users   LIMIT 2`
	expect = strings.TrimSpace(expect)
	if sql.Q != expect {
		t.Errorf("expected %s got %s", expect, sql.Q)
	}
}

func TestDB_Find(t *testing.T) {
	db, err := Open("ql-mem", "test.db")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = db.Close() }()
	_, err = db.Automigrate(&Foo{})
	if err != nil {
		t.Fatal(err)
	}

	sample := []string{"a", "b", "c", "d"}
	for _, v := range sample {
		err := db.Create(&Foo{Stuff: v})
		if err != nil {
			t.Fatal(err)
		}
	}
	fu := []Foo{}
	err = db.Find(&fu)
	if err != nil {
		t.Fatal(err)
	}
	if len(fu) != 4 {
		t.Errorf("expected 4 got  %d", len(fu))
	}
}

func TestDB_FirstOrInit(t *testing.T) {
	db, err := Open("ql-mem", "test.db")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = db.Close() }()
	_, err = db.Automigrate(&Foo{})
	if err != nil {
		t.Fatal(err)
	}
	fu := Foo{}
	err = db.Where(Foo{Stuff: "nah"}).Attrs(Foo{ID: 20}).FirstOrInit(&fu)
	if err != nil {
		t.Fatal(err)
	}
	if fu.ID != 20 {
		t.Errorf("expected 20 got %d", fu.ID)
	}
}

func TestDB_Save(t *testing.T) {
	db, err := Open("ql-mem", "test.db")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = db.Close() }()
	_, err = db.Automigrate(&Foo{})
	if err != nil {
		t.Fatal(err)
	}

	sample := []string{"a", "b", "c", "d"}
	for _, v := range sample {
		err := db.Create(&Foo{Stuff: v})
		if err != nil {
			t.Fatal(err)
		}
	}
	fu := Foo{}
	err = db.Begin().First(&fu)
	if err != nil {
		t.Fatal(err)
	}
	fu.Stuff = "updates"
	err = db.Save(&fu)
	if err != nil {
		t.Fatal(err)
	}
	first := Foo{}
	err = db.Begin().First(&first)
	if err != nil {
		t.Fatal(err)
	}
	if first.Stuff != fu.Stuff {
		t.Errorf("expected %s got %s", fu.Stuff, first.Stuff)
	}
}

func TestDB_Update(t *testing.T) {
	db, err := Open("ql-mem", "test.db")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = db.Close() }()
	_, err = db.Automigrate(&Foo{})
	if err != nil {
		t.Fatal(err)
	}

	sample := []string{"a", "b", "c", "d"}
	for _, v := range sample {
		err := db.Create(&Foo{Stuff: v})
		if err != nil {
			t.Fatal(err)
		}
	}
	fu := Foo{}
	err = db.Begin().First(&fu)
	if err != nil {
		t.Fatal(err)
	}
	up := "stuff"
	err = db.Begin().Model(&fu).Update("stuff", up)
	if err != nil {
		t.Fatal(err)
	}
	first := Foo{}
	err = db.Begin().First(&first)
	if err != nil {
		t.Fatal(err)
	}
	if first.Stuff != up {
		t.Errorf("expected %s got %s", up, first.Stuff)
	}
}

func TestDB_Assign(t *testing.T) {
	db, err := Open("ql-mem", "test.db")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = db.Close() }()
	_, err = db.Automigrate(&fixture.User{})
	if err != nil {
		t.Fatal(err)
	}
	user := fixture.User{}
	err = db.Where(
		fixture.User{Name: "non_existing"}).Assign(
		fixture.User{Age: 20}).FirstOrInit(&user)
	if err != nil {
		t.Fatal(err)
	}
	if user.Age != 20 {
		t.Errorf("expected 40 got %d", user.Age)
	}

}

func TestDB_Pluck(t *testing.T) {
	db, err := Open("ql-mem", "test.db")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = db.Close() }()
	_, err = db.Automigrate(&Foo{})
	if err != nil {
		t.Fatal(err)
	}

	sample := []string{"a", "b", "c", "d"}
	for _, v := range sample {
		err := db.Create(&Foo{Stuff: v})
		if err != nil {
			t.Fatal(err)
		}
	}
	var stuffs []string
	err = db.Begin().Model(&Foo{}).Pluck("stuff", &stuffs)
	if err != nil {
		t.Fatal(err)
	}
	if len(stuffs) != 4 {
		t.Errorf("expected %d got %d", 4, len(stuffs))
	}
}

func TestDB_Count(t *testing.T) {
	db, err := Open("ql-mem", "test.db")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = db.Close() }()
	_, err = db.Automigrate(&Foo{})
	if err != nil {
		t.Fatal(err)
	}

	sample := []string{"a", "b", "c", "d"}
	for _, v := range sample {
		err := db.Create(&Foo{Stuff: v})
		if err != nil {
			t.Fatal(err)
		}
	}
	var stuffs int64
	err = db.Begin().Model(&Foo{}).Count(&stuffs)
	if err != nil {
		t.Fatal(err)
	}
	if stuffs != 4 {
		t.Errorf("expected %d got %d", 4, stuffs)
	}
}

func TestDB_AddIndexSQL(t *testing.T) {
	db, err := Open("ql-mem", "test.db")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = db.Close() }()
	_, err = db.Automigrate(&Foo{})
	if err != nil {
		t.Fatal(err)
	}
	_, err = db.AddIndexSQL("_idx_foo_stuff", "stuff")
	if err == nil {
		t.Fatal("expected an error")
	}

	sql, err := db.Model(&Foo{}).AddIndexSQL("_idx_foo_stuff", "stuff")
	if err != nil {
		t.Fatal(err)
	}
	expect := `CREATE INDEX _idx_foo_stuff ON foos(stuff) `
	if sql.Q != expect {
		t.Errorf("expected %s got %s", expect, sql.Q)
	}
}

func TestDB_AddIndex(t *testing.T) {
	db, err := Open("ql-mem", "test.db")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = db.Close() }()
	_, err = db.Automigrate(&Foo{})
	if err != nil {
		t.Fatal(err)
	}
	i := "idx_foo_stuff"
	_, err = db.Model(&Foo{}).AddIndex(i, "stuff")
	if err != nil {
		t.Fatal(err)
	}
	if !db.Dialect().HasIndex("foos", i) {
		t.Error("expected index to be created")
	}
}

func TestDB_DeleteSQL(t *testing.T) {
	db, err := Open("ql-mem", "test.db")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = db.Close() }()
	_, err = db.Automigrate(&Foo{})
	if err != nil {
		t.Fatal(err)
	}
	sql, err := db.DeleteSQL(&Foo{ID: 10})
	if err != nil {
		t.Fatal(err)
	}
	expect := `
BEGIN TRANSACTION;
	DELETE FROM foos  WHERE id = $1 ;
COMMIT;
`
	expect = strings.TrimSpace(expect)
	sql.Q = strings.TrimSpace(sql.Q)
	if sql.Q != expect {
		t.Errorf("expected %s got %s", expect, sql.Q)
	}
}

func TestDB_Delete(t *testing.T) {
	db, err := Open("ql-mem", "test.db")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = db.Close() }()
	_, err = db.Automigrate(&Foo{})
	if err != nil {
		t.Fatal(err)
	}
	f := Foo{Stuff: "halloween"}
	err = db.Create(&f)
	if err != nil {
		t.Fatal(err)
	}
	if f.ID == 0 {
		t.Fatalf("expected a new record to be created")
	}

	err = db.Delete(&f)
	if err != nil {
		t.Fatal(err)
	}
	fu := Foo{}
	err = db.Model(&Foo{ID: f.ID}).First(&fu)
	if err == nil {
		t.Fatal("expected an error")
	}
}

func TestDB_AddUniqueIndex(t *testing.T) {
	db, err := Open("ql-mem", "test.db")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = db.Close() }()
	_, err = db.Automigrate(&Foo{})
	if err != nil {
		t.Fatal(err)
	}
	i := "idx_foo_stuff"
	ndb := db.Model(&Foo{})
	_, err = ndb.AddUniqueIndex(i, "stuff")
	if err != nil {
		t.Fatal(err)
	}
	if !db.Dialect().HasIndex("foos", i) {
		t.Error("expected index to be created")
	}
	q := ndb.e.Scope.SQL
	expect := `CREATE UNIQUE INDEX idx_foo_stuff ON foos(stuff) `
	if q != expect {
		t.Errorf("expected %s got %s", expect, q)
	}
}
func TestDB_RemoveIndex(t *testing.T) {
	db, err := Open("ql-mem", "test.db")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = db.Close() }()
	_, err = db.Automigrate(&Foo{})
	if err != nil {
		t.Fatal(err)
	}
	i := "idx_foo_stuff"
	ndb := db.Model(&Foo{})
	_, err = ndb.AddUniqueIndex(i, "stuff")
	if err != nil {
		t.Fatal(err)
	}
	if !db.Dialect().HasIndex("foos", i) {
		t.Error("expected index to be created")
	}
	err = db.Model(&Foo{}).RemoveIndex(i)
	if err != nil {
		t.Fatal(err)
	}
	if db.Dialect().HasIndex("foos", i) {
		t.Error("expected index to be gone")
	}
}
func TestDB_DropColumn(t *testing.T) {
	db, err := Open("ql-mem", "test.db")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = db.Close() }()
	_, err = db.Automigrate(&Foo{})
	if err != nil {
		t.Fatal(err)
	}
	col := "stuff"
	_, err = db.Model(&Foo{}).DropColumn(col)
	if err != nil {
		t.Fatal(err)
	}
	if db.Dialect().HasColumn("foos", col) {
		t.Error("expected column to be gone")
	}
}

func TestDB_FirstOrCreate(t *testing.T) {
	db, err := Open("ql-mem", "test.db")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = db.Close() }()
	_, err = db.Automigrate(&Foo{})
	if err != nil {
		t.Fatal(err)
	}
	first := Foo{Stuff: "first"}
	err = db.FirstOrCreate(&first)
	if err != nil {
		t.Fatal(err)
	}
	if first.ID == 0 {
		t.Error("expected a new record")
	}
	second := Foo{}
	err = db.Begin().Where(Foo{Stuff: first.Stuff}).FirstOrCreate(&second)
	if err != nil {
		t.Fatal(err)
	}
	if second.ID != first.ID {
		t.Errorf("expected %d got %d", first.ID, second.ID)
	}
}
