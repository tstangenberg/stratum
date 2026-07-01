# GraphQL Beispiele

Alle Queries werden per `POST /graphql/{schema-name}` ausgeführt.

Das Beispiel-Schema für diese Queries:

```graphql
type Location {
  id: ID!
  name: String!
  city: String!
}
```

---

## List

Alle Einträge abrufen:

```graphql
{
  location {
    list {
      id
      name
      city
    }
  }
}
```

Mit Filter:

```graphql
{
  location {
    list(filter: { city: { eq: "Berlin" } }) {
      id
      name
      city
    }
  }
}
```

Mit Pagination:

```graphql
{
  location {
    list(limit: 10, offset: 0) {
      id
      name
      city
    }
  }
}
```

---

## Get

Einzelnen Eintrag per ID abrufen:

```graphql
{
  location {
    get(id: "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx") {
      id
      name
      city
    }
  }
}
```

---

## Create

```graphql
mutation {
  location {
    create(input: {
      name: "Brandenburger Tor"
      city: "Berlin"
    }) {
      id
      name
      city
    }
  }
}
```

Mit expliziter ID:

```graphql
mutation {
  location {
    create(input: {
      id: "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
      name: "Brandenburger Tor"
      city: "Berlin"
    }) {
      id
      name
      city
    }
  }
}
```
