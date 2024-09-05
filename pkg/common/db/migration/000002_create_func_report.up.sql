create or replace function report(target_period daterange) returns table (car varchar(10), count_of_days bigint)
as $$
declare
r rent%rowtype;
begin

	return query select
		tt.gos_num, count(tt.pp)
		from (
			select tmp.gos_num, generate_series(lower(tmp.p),upper(tmp.p)- interval '1 day', interval '1 day') as pp
			from (
					select gos_num, (target_period * "period") as p
					from rent r
					)as tmp
			) as tt
		group by tt.gos_num;
end;
$$ language plpgsql;