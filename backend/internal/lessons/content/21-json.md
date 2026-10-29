title: JSON columns
module: Dates, text & JSON
level: Intermediate
minutes: 12
summary: Read values out of JSON documents, filter on them and build JSON in queries.
---
`products.attributes` is a `JSON` column holding flexible details that vary by product type:

```json
{"color": "black", "warranty_years": 2, "weight_kg": 0.31, "wireless": true}
{"sizes": ["S", "M", "L"], "material": "cotton", "color": "navy"}
{"pages": 412, "format": "hardcover", "language": "English"}
```

### Extracting values

| Syntax | Returns |
|---|---|
| `attributes->'$.color'` | JSON value: `"black"` (with quotes) |
| `attributes->>'$.color'` | unquoted text: `black` |
| `JSON_EXTRACT(attributes, '$.sizes[0]')` | first array element |
| `JSON_UNQUOTE(JSON_EXTRACT(...))` | same as `->>` |

Paths start with `$` (the document root). Use `.key` for object keys and `[n]` for array positions.

```sql
SELECT product_id, name, attributes->>'$.color' AS color
FROM products
WHERE category_id = 42;
```

### Filtering

```sql
-- booleans come out as the text 'true' / 'false' with ->>
SELECT COUNT(*) FROM products WHERE attributes->>'$.wireless' = 'true';

-- does an array contain a value?
SELECT COUNT(*) FROM products WHERE JSON_CONTAINS(attributes->'$.sizes', '"XL"');

-- does a key exist?
SELECT COUNT(*) FROM products WHERE JSON_CONTAINS_PATH(attributes, 'one', '$.pages');
```

### Building JSON

`JSON_OBJECT(key, value, ...)`, `JSON_ARRAY(...)` and the aggregates `JSON_ARRAYAGG(x)` / `JSON_OBJECTAGG(k, v)` produce JSON. That's handy for APIs:

```sql
SELECT JSON_OBJECT('department', d.name, 'headcount', COUNT(*)) AS doc
FROM employees e JOIN departments d USING (department_id)
GROUP BY d.name;
```

> JSON is great for sparse, varying attributes. If you filter or join on a field all the time, it probably deserves a real column (or a generated column with an index).

=== exercise
id: json-1
title: Headphone colors
--- prompt
For products in category **42** (Headphones), return `product_id`, `name` and **color**, taken from the `attributes` JSON as plain text (no quotes).
--- solution
SELECT product_id, name, attributes->>'$.color' AS color
FROM products
WHERE category_id = 42;
--- hint
attributes->>'$.color' unquotes the value.
===

=== exercise
id: json-2
title: Wireless by category
ordered: true
--- prompt
Count **wireless** products (`wireless` is `true` in `attributes`) per category: return the category **name** and **products**, most first.
--- solution
SELECT c.name, COUNT(*) AS products
FROM products p
JOIN categories c ON c.category_id = p.category_id
WHERE p.attributes->>'$.wireless' = 'true'
GROUP BY c.category_id, c.name
ORDER BY products DESC, c.name;
--- hint
Compare attributes->>'$.wireless' with the text 'true'.
===

=== exercise
id: json-3
title: Available in XL
--- prompt
How many products list **XL** among their `sizes`? Return the count as **products**.
--- solution
SELECT COUNT(*) AS products
FROM products
WHERE JSON_CONTAINS(attributes->'$.sizes', '"XL"');
--- hint
JSON_CONTAINS(attributes->'$.sizes', '"XL"'). The candidate must itself be valid JSON, hence the double quotes.
===

=== exercise
id: json-4
title: Book stats as JSON
ordered: true
--- prompt
For each book `format` (from the `attributes` of products that have a `pages` key), return **format** and a JSON document **stats** with keys `books` (count) and `avg_pages` (average pages rounded to 0 decimals). Sort by format.
--- solution
SELECT attributes->>'$.format' AS format,
       JSON_OBJECT('books', COUNT(*), 'avg_pages', ROUND(AVG(attributes->>'$.pages'), 0)) AS stats
FROM products
WHERE JSON_CONTAINS_PATH(attributes, 'one', '$.pages')
GROUP BY format
ORDER BY format;
--- hint
JSON_OBJECT('books', COUNT(*), 'avg_pages', ROUND(AVG(attributes->>'$.pages'), 0))
===
