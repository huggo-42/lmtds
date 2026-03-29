# lmtds - laravel migrations to dbdiagram schema

## Build
```console
$ go build
```

## Run
```console
$ ./lmtds generate-m migrations -o dbdiagram.dbml
```

<br>

## Example

- Given a directory migrations with a migration 2042_04_20_1729_create_table_user.php
├── migrations
│   ├── 2042_04_20_1729_create_table_user.php

- With the migration being
```2042_04_20_1729_create_table_user.php
public function up()
{
    Schema::create('user', function (Blueprint $table) {
        $table->increments('id');
        $table->string('username');
        $table->string('role');
        $table->enum('gender', ['F', 'M'])->comment('F => female, M => men');
        $table->timestamp('created_at');
    });
}
```

- The output in dbdiagram.dbml is
```dbdiagram.dbml
Table user {
   id integer [primary key]
   username varchar
   role varchar
   gender enum [not null, note: 'F => female, M => men']
   created_at timestamp [not null]
}
```
