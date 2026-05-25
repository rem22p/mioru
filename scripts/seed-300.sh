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
  
  case $cid in
    2|3|4|5|6|7|8|9|10)
      sz=$(rand "${SIZES_CLOTHING[@]}")","$(rand "${SIZES_CLOTHING[@]}")","$(rand "${SIZES_CLOTHING[@]}")
      ;;
    12|13|14)
      sz=$(rand "${SIZES_SHOES[@]}")","$(rand "${SIZES_SHOES[@]}")","$(rand "${SIZES_SHOES[@]}")","$(rand "${SIZES_SHOES[@]}")
      ;;
    *) sz="One size" ;;
  esac
  
  slug="${brand,,}-${cname,,}-${i}"
  slug="${slug//\//-}"
  slug="${slug// /-}"
  
  name="$brand $cname #${i}"
  
  echo "INSERT INTO products (slug, category_id, brand, name, price, color, material, description, xp_reward, in_stock, status, stock_quantity, created_by)"
  echo "VALUES ('$slug', $cid, '$brand', '$name', $price, '$color', 'Премиальный материал', 'Качественный товар от $brand. Современный дизайн и комфорт на каждый день.', $((price/100)), 1, 'in_stock', $((RANDOM % 30 + 1)), 'seed');"
  
  # Insert sizes via subquery
  echo "INSERT INTO product_sizes (product_id, size_label) VALUES (currval(pg_get_serial_sequence('products','id')), unnest(ARRAY['$sz']));"
done
echo "COMMIT;"
