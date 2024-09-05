create table if not exists car_park(
    id serial primary key check (id between 1 and 5),
    gos_num varchar(10) unique not null
    );

create table if not exists rent (
    id serial primary key,
    gos_num varchar(10) not null references car_park (gos_num) ON DELETE CASCADE,
    period daterange not null,
    cost real not null
    );