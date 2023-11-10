package db

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
	"unicode"

	"gorm.io/gorm"
)

type With struct {
	Where *Where           `json:"where,omitempty"`
	With  map[string]*With `json:"with,omitempty"`
}

type Where struct {
	Not   *Where   `json:"not,omitempty"`
	Or    []*Where `json:"or,omitempty"`
	And   []*Where `json:"and,omitempty"`
	Field *Field   `json:"field,omitempty"`
}

type Field struct {
	Name      string `json:"name,omitempty"`
	Predicate string `json:"predicate,omitempty"`
	Value     any    `json:"value,omitempty"`
}

type Order struct {
	Field     string `json:"field,omitempty"`
	Direction string `json:"direction,omitempty"`
}

type Query struct {
	Select   []string          `json:"select,omitempty"`
	Omit     []string          `json:"omit,omitempty"`
	With     map[string]*With  `json:"with,omitempty"`
	Limit    *uint             `json:"limit,omitempty"`
	Offset   *uint             `json:"offset,omitempty"`
	Where    *Where            `json:"where,omitempty"`
	Orders   []Order           `json:"orders,omitempty"`
	Preloads map[string]*Query `json:"preloads,omitempty"`
}

func (q *Query) P(client *gorm.DB, table string) (*gorm.DB, error) {
	if client == nil {
		return nil, errors.New("query: gorm client is nil")
	}

	var count uint

	if len(q.Preloads) > 0 {
		relations, ok := edgesMap[table]
		if !ok {
			return nil, fmt.Errorf("query: cannot preload %s", table)
		}

		for key, value := range q.Preloads {
			relation, ok := relations[key]
			if !ok {
				return nil, fmt.Errorf("query: invalid relation %s", key)
			}

			if value == nil {
				client = client.Preload(edge(key))
			} else {
				client = client.Preload(edge(key), func(db *gorm.DB) *gorm.DB {
					ndb, err := value.P(db, relation[0])
					if err != nil {
						return db
					}
					return ndb
				})
			}

		}
	}

	prefix := client.NamingStrategy.TableName("")
	if len(q.With) > 0 {
		for key, value := range q.With {
			if _, ok := edgesMap[table][key]; !ok {
				return nil, fmt.Errorf("query: invalid with relation %s", key)
			}
			join, query, vars, err := value.P(count, prefix, table, edgesMap[table][key])
			if err != nil {
				return nil, err
			}
			if len(query) > 0 {
				client = client.InnerJoins(strings.Join([]string{join, query}, " AND "), vars...)
			} else {
				client = client.InnerJoins(join, vars...)
			}
		}
	}

	if len(q.Select) > 0 {
		client = client.Select(q.Select)
	}

	if q.Where != nil {
		query, variables, err := q.Where.P(prefix + table + ".")
		if err != nil {
			return nil, err
		}
		if query != "" {
			client = client.Where(query, variables...)
		}
	}

	if q.Limit != nil {
		client = client.Limit(int(*q.Limit))
	}

	if q.Offset != nil {
		client = client.Offset(int(*q.Offset))
	}

	if len(q.Orders) > 0 {
		for _, order := range q.Orders {
			if !isField(order.Field) {
				return nil, fmt.Errorf("order: field %s is not alphanumeric", order.Field)
			}
			order.Field = `"` + prefix + table + `"."` + order.Field + `"`

			if order.Direction == "" {
				order.Direction = "ASC"
			} else if strings.ToUpper(order.Direction) != "ASC" && strings.ToUpper(order.Direction) != "DESC" {
				return nil, fmt.Errorf("order: direction for field %s must be ASC or DESC", order.Field)
			}
			client = client.Order(order.Field + " " + order.Direction)
		}
	}

	return client, nil
}

func (w *With) P(count uint, prefix, table string, relation []string) (string, string, []any, error) {

	var err error
	var vars []any
	var where, join string
	count++

	newPrefix := fmt.Sprintf("%s_%d", prefix+relation[0], count)

	if len(relation) == 3 {
		join = fmt.Sprintf(
			`INNER JOIN "%s" as "%s" ON "%s" = "%s"`,
			prefix+relation[0],
			newPrefix,
			prefix+table+`"."`+relation[1],
			newPrefix+`"."`+relation[2],
		)
	}

	if len(relation) == 6 {
		midPrefix := fmt.Sprintf("%s_%d", prefix+relation[3], count)
		join = fmt.Sprintf(
			`INNER JOIN "%s" as "%s" ON "%s" = "%s" INNER JOIN "%s" as "%s" ON "%s" = "%s"`,
			prefix+relation[3],
			midPrefix,
			prefix+table+`"."`+relation[1],
			midPrefix+`"."`+relation[4],
			prefix+relation[0],
			newPrefix,
			newPrefix+`"."`+relation[1],
			midPrefix+`"."`+relation[5],
		)
	}

	if w.Where != nil {
		where, vars, err = w.Where.P(newPrefix + `.`)
	}

	if len(w.With) > 0 {
		table = relation[0]
		for key, with := range w.With {
			if relation, ok := edgesMap[relation[0]][key]; ok {
				subJoin, subWhere, subVars, subErr := with.P(count, prefix, table, relation)
				if subErr != nil {
					return "", "", nil, fmt.Errorf("with: cannot find relation %s for table %s", key, table)
				}
				if len(subJoin) > 0 {
					join += " " + subJoin
				}
				if len(subWhere) > 0 {
					where += " AND " + subWhere
					vars = append(subVars, vars)
				}
			} else {
				return "", "", nil, fmt.Errorf("with: cannot find relation %s for table %s", key, table)
			}
		}
	}

	return join, where, vars, err
}

func (w *Where) P(prefixes ...string) (string, []any, error) {
	var queries []string
	var variables []any
	var prefix string

	if len(prefixes) > 0 {
		prefix = prefixes[0]
	}

	if w.Not != nil {
		q, v, err := w.Not.P(prefix)
		if err != nil {
			return "", nil, err
		}
		queries = append(queries, fmt.Sprintf("NOT ( %s )", q))
		variables = append(variables, v)
	}

	if len(w.And) > 0 {
		var andQueries []string
		var andVariables []any

		for _, p := range w.And {
			q, vs, err := p.P(prefix)
			if err != nil {
				return "", nil, err
			}
			andQueries = append(andQueries, q)
			andVariables = append(andVariables, vs...)
		}

		queries = append(queries, strings.Join(andQueries, " AND "))
		variables = append(variables, andVariables...)
	}

	if len(w.Or) > 0 {
		var orQueries []string
		var orVariables []any

		for _, p := range w.Or {
			q, vs, err := p.P(prefix)
			if err != nil {
				return "", nil, err
			}
			orQueries = append(orQueries, q)
			orVariables = append(orVariables, vs...)
		}

		if len(w.Or) == 1 {
			queries = append(queries, orQueries...)
		}

		if len(w.Or) == 2 {
			queries = append(queries, "("+strings.Join(orQueries, " OR ")+")")
		}

		variables = append(variables, orVariables...)
	}

	if w.Field != nil {
		var fieldQuery string
		field := w.Field.Name
		if !isField(w.Field.Name) {
			return "", nil, fmt.Errorf("where: %+v has to be a valid field", field)
		}
		field = strings.ReplaceAll(prefix+w.Field.Name, `.`, `"."`)
		predicate := w.Field.Predicate
		switch predicate {
		case "like":
			fieldQuery = fmt.Sprintf(`"%s" LIKE (?)`, field)
			variables = append(variables, w.Field.Value)
		case "null":
			fieldQuery = fmt.Sprintf(`"%s" IS NULL`, field)
		case "not null":
			fieldQuery = fmt.Sprintf(`"%s" IS NOT NULL`, field)
		case "between":
			fieldQuery = fmt.Sprintf(`"%s" BETWEEN ? AND ?`, field)
			value := w.Field.Value.([]any)
			variables = append(variables, value[:]...)
		case "in":
			fieldQuery = fmt.Sprintf(`"%s" IN (?)`, field)
			variables = append(variables, w.Field.Value)
		case "not in":
			fieldQuery = fmt.Sprintf(`"%s" NOT IN (?)`, field)
			variables = append(variables, w.Field.Value)
		case "=":
			fieldQuery = fmt.Sprintf(`"%s" = ?`, field)
			variables = append(variables, w.Field.Value)
		case "<>":
			fieldQuery = fmt.Sprintf(`"%s" <> ?`, field)
			variables = append(variables, w.Field.Value)
		case ">":
			fieldQuery = fmt.Sprintf(`"%s" > ?`, field)
			variables = append(variables, w.Field.Value)
		case ">=":
			fieldQuery = fmt.Sprintf(`"%s" >= ?`, field)
			variables = append(variables, w.Field.Value)
		case "<":
			fieldQuery = fmt.Sprintf(`"%s" < ?`, field)
			variables = append(variables, w.Field.Value)
		case "<=":
			fieldQuery = fmt.Sprintf(`"%s" <= ?`, field)
			variables = append(variables, w.Field.Value)
		default:
			return "", nil, fmt.Errorf("where: %+v invalid predicate", predicate)
		}
		queries = append(queries, fieldQuery)
	}
	return strings.Join(queries, " AND "), variables, nil
}

func isField(field string) bool {
	_, err := regexp.MatchString(`^\w+(\.\w+)*$`, field)
	return err == nil
}

func edge(s string) string {
	return string(unicode.ToUpper(rune(s[0]))) + s[1:]
}

var edgesMap = map[string]map[string][]string{
	"users": {},
}
