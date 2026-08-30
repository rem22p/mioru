#!/bin/bash
# Generate SQL for 300 test products
BRANDS=("Nike" "Adidas" "Puma" "Reebok" "New Balance" "Asics" "Vans" "Lacoste" "Champion" "Fila")
COLORS=("Чёрный" "Белый" "Серый" "Синий" "Красный" "Зелёный" "Бежевый" "Коричневый" "Оливковый" "Бордовый")
SIZES_CLOTHING=("XS" "S" "M" "L" "XL" "XXL")
SIZES_SHOES=("40" "41" "42" "43" "44" "45")

# Category distribution: mostly clothing (60%), shoes (25%), accessories (15%)
cat_ids=(2 2 2 2 3 3 3 4 4 4 4 5 5 6 6 7 7 8 8 9 10 12 12 12 12 12 13 13 14 14 15 17 19 24)
cat_names=(
  "Футболки / поло" "Футболки / поло" "Футболки / поло" "Футболки / поло"
  "Шорты" "Шорты" "Шорты"
  "Худи / зип-худи" "Худи / зип-худи" "Худи / зип-худи" "Худи / зип-худи"
  "Свитшоты / свитера" "Свитшоты / свитера"
  "Джинсы" "Джинсы"
  "Штаны" "Штаны"
  "Куртки" "Куртки"
  "Жилетки" "Нижнее бельё"
  "Кроссовки" "Кроссовки" "Кроссовки" "Кроссовки" "Кроссовки"
  "Тапки" "Тапки"
  "Ботинки" "Ботинки"
  "Сумки"
  "Кошельки / кардхолдеры"
  "Головные уборы"
  "Часы"
)

rand() { local arr=("$@"); echo "${arr[$((RANDOM % ${#arr[@]}))]}"; }
rprice() { echo $(( (RANDOM % 150 + 1) * 100 )); }

echo "BEGIN;"
for i in $(seq 1 300); do
  idx=$((RANDOM % ${#cat_ids[@]}))
  cid=${cat_ids[$idx]}
  cname="${cat_names[$idx]}"
  brand=$(rand "${BRANDS[@]}")
  color=$(rand "${COLORS[@]}")
  price=$(rprice)
  
  # Pick sizes as a bash array so they survive into SQL as separate rows
  # (the old comma-joined string ended up as ARRAY['40,41,42,43'] — one
  # element, so unnest produced a single bogus "40,41,42,43" size_label).
  case $cid in
    2|3|4|5|6|7|8|9|10)
      sizes=("$(rand "${SIZES_CLOTHING[@]}")" "$(rand "${SIZES_CLOTHING[@]}")" "$(rand "${SIZES_CLOTHING[@]}")")
      ;;
    12|13|14)
      sizes=("$(rand "${SIZES_SHOES[@]}")" "$(rand "${SIZES_SHOES[@]}")" "$(rand "${SIZES_SHOES[@]}")" "$(rand "${SIZES_SHOES[@]}")")
      ;;
    *) sizes=("One size") ;;
  esac

  # Dedupe — rand can hit the same value twice and a real product never
  # carries the same size_label twice.
  declare -A seen=()
  unique_sizes=()
  for s in "${sizes[@]}"; do
    if [[ -z "${seen[$s]+x}" ]]; then
      seen[$s]=1
      unique_sizes+=("$s")
    fi
  done

  # Render ARRAY['40','41','42'] — quoted, comma-joined.
  sz_sql=""
  for s in "${unique_sizes[@]}"; do
    sz_sql+="'${s//\'/\'\'}',"
  done
  sz_sql="${sz_sql%,}"

  slug="${brand,,}-${cname,,}-${i}"
  slug="${slug//\//-}"
  slug="${slug// /-}"

  name="$brand $cname #${i}"

  echo "INSERT INTO products (slug, category_id, brands, name, price, color, material, description, xp_reward, in_stock, status, stock_quantity, created_by)"
  echo "VALUES ('$slug', $cid, ARRAY['$brand'], '$name', $price, '$color', 'Премиальный материал', 'Качественный товар от $brand. Современный дизайн и комфорт на каждый день.', $((price/100)), 1, 'in_stock', $((RANDOM % 30 + 1)), 'seed');"

  echo "INSERT INTO product_sizes (product_id, size_label) VALUES (currval(pg_get_serial_sequence('products','id')), unnest(ARRAY[$sz_sql]));"
done
echo "COMMIT;"
