# Expense Manager — API Documentation

Complete reference for the frontend team to integrate with the Expense Manager backend (Go implementation).

---

## Table of Contents

1. [Base Information](#base-information)
2. [Authentication](#authentication)
3. [Auth Endpoints](#auth-endpoints)
4. [Users](#users)
5. [Categories](#categories)
6. [Groups](#groups)
7. [Group Members](#group-members)
8. [Expenses](#expenses)
9. [Balances](#balances)
10. [Settlements](#settlements)
11. [Data Types & Conventions](#data-types--conventions)
12. [Error Responses](#error-responses)

---

## Base Information

| Item | Value |
|------|-------|
| Base URL | `http://<host>:8080` (e.g. `http://localhost:8080`) |
| Content Type | `application/json` |
| Authentication | Bearer JWT (see [Authentication](#authentication)) |
| Health check | `GET /health` |

All request/response bodies are JSON. IDs are UUIDs serialized as strings. Monetary amounts are JSON **numbers** rounded to two decimals (e.g. `150.5`, `33.34`). Datetimes are RFC 3339 strings.

Routes have **no `/api` prefix** — e.g. `POST /auth/login`, `GET /groups`.

---

## Authentication

Most endpoints require authentication. Send the access token in the `Authorization` header:

```
Authorization: Bearer <access_token>
```

### Token Types

| Token | Purpose | Expiry |
|-------|---------|--------|
| `access_token` | Authenticate API requests | 30 minutes |
| `refresh_token` | Obtain a new access token | 7 days |

### Auth Flow

1. Register or login to get `access_token` + `refresh_token`.
2. Send `access_token` as Bearer token on protected requests.
3. When the access token expires, call `POST /auth/refresh` with the `refresh_token` to get a new pair.

**Public endpoints (no auth required):**
- `POST /auth/register`
- `POST /auth/login`
- `POST /auth/refresh`
- `POST /users`

All other endpoints require a valid access token. The signing secret is read from the `JWT_SECRET` environment variable (a development default with a startup warning is used when unset).

---

## Auth Endpoints

### `POST /auth/register`

Register a new user. **No auth required.**

**Request Body:**

```json
{
  "email": "user@example.com",
  "password": "secret123",
  "full_name": "John Doe"
}
```

| Field | Type | Required | Notes |
|-------|------|----------|-------|
| `email` | string | ✅ | Valid email format, unique. Trimmed and lowercased server-side — uniqueness is **case-insensitive** (`User@Example.com` and `user@example.com` are the same account) |
| `password` | string | ✅ | Plain text (hashed server-side) |
| `full_name` | string | ❌ | Optional |

**Response `201 Created`:**

```json
{
  "id": "3fa85f64-5717-4562-b3fc-2c963f66afa6",
  "email": "user@example.com",
  "full_name": "John Doe",
  "is_active": true,
  "created_at": "2026-08-16T00:00:00Z",
  "updated_at": null
}
```

**Errors:**
- `400` — `email already registered`
- `400` — `email must be a valid email address`
- `400` — `email is required` / `password is required`

---

### `POST /auth/login`

Authenticate and receive tokens. **No auth required.**

**Request Body:**

```json
{
  "email": "user@example.com",
  "password": "secret123"
}
```

Email matching is case-insensitive (the address is trimmed and lowercased before lookup).

**Response `200 OK`:**

```json
{
  "access_token": "eyJhbGciOi...",
  "refresh_token": "eyJhbGciOi...",
  "token_type": "bearer"
}
```

**Errors:**
- `401` — `invalid email or password`
- `403` — `account is deactivated`

---

### `POST /auth/refresh`

Exchange a refresh token for a new token pair. **No auth required.**

**Request Body:**

```json
{
  "refresh_token": "eyJhbGciOi..."
}
```

**Response `200 OK`:** Same token pair shape as login.

**Errors:**
- `401` — `invalid token` / `invalid or expired token` / `invalid token type` (an access token was supplied) / `User not found or inactive`

---

### `GET /auth/me`

Get the currently authenticated user. **Auth required.**

**Response `200 OK`:** Single user object (see register response).

**Errors:**
- `401` — `User not found`

---

## Users

User object shape (password is never serialized):

```json
{
  "id": "3fa85f64-...",
  "email": "user@example.com",
  "full_name": "John Doe",
  "is_active": true,
  "created_at": "2026-08-16T00:00:00Z",
  "updated_at": null
}
```

### `GET /users`

List all users. **Auth required.** → `200 OK` with a JSON array of user objects.

---

### `GET /users/{user_id}`

Get a single user by ID. **Auth required.**

**Errors:**
- `404` — `user not found`

---

### `POST /users`

Create a user. **No auth required.** Same body and behavior as `POST /auth/register`.

---

### `PUT /users/{user_id}`

Update a user. **Auth required.** All fields optional; only provided fields are updated.

**Request Body:**

```json
{
  "email": "new@example.com",
  "full_name": "Jane Doe",
  "password": "newpass123",
  "is_active": true
}
```

| Field | Type | Required | Notes |
|-------|------|----------|-------|
| `email` | string | ❌ | Must be unique (case-insensitive; trimmed and lowercased server-side) |
| `full_name` | string | ❌ | |
| `password` | string | ❌ | Re-hashed server-side |
| `is_active` | boolean | ❌ | |

**Response `200 OK`:** Updated user object (`updated_at` is set).

**Errors:**
- `404` — `user not found`
- `400` — `email already registered` / `email must be a valid email address`

---

### `DELETE /users/{user_id}`

Delete a user. **Auth required.**

**Response `200 OK`:** `"User deleted successfully!"`

**Errors:**
- `404` — `user not found`

---

## Categories

Categories are **group-scoped**: every route carries a `{group_id}`, and category names are unique per group (`UNIQUE(group_id, name)`). Deleting a category that has expenses is blocked by a foreign key restriction.

Category object shape:

```json
{
  "id": "3fa85f64-...",
  "group_id": "3fa85f64-...",
  "name": "Food",
  "is_active": true,
  "created_at": "2026-08-16T00:00:00Z",
  "updated_at": "2026-08-16T00:00:00Z"
}
```

### `POST /categories`

Create a category in a group. **Auth required.**

**Request Body:**

```json
{
  "group_id": "3fa85f64-...",
  "name": "Food"
}
```

**Response `201 Created`:** Single category object.

**Errors:**
- `400` — `group_id and name are required`
- `400` — `Category already exists!`
- `404` — `group not found`

---

### `GET /categories/{group_id}`

List a group's categories. **Auth required.** → `200 OK` with a JSON array.

---

### `GET /categories/{group_id}/{id}`

Get a single category. **Auth required.**

**Errors:**
- `404` — `category not found`
- `403` — `invalid access` (category belongs to a different group)

---

### `PUT /categories/{group_id}/{id}`

Update a category. **Auth required.**

**Request Body:**

```json
{
  "name": "Groceries",
  "is_active": true
}
```

Both fields are required.

**Response `200 OK`:** Updated category object.

**Errors:**
- `400` — `name and is_active are required` / `Category already exists!`
- `404` — `category not found`

---

### `DELETE /categories/{group_id}/{id}`

Delete a category. **Auth required.**

**Response `200 OK`:** `"Category Food deleted successfully!"`

**Errors:**
- `404` — `category not found`

---

## Groups

Group object shape (members are included in all group responses):

```json
{
  "id": "3fa85f64-...",
  "name": "Trip to Goa",
  "created_by": "3fa85f64-...",
  "simplify_debts": true,
  "currency": "INR",
  "is_active": true,
  "created_at": "2026-08-16T00:00:00Z",
  "updated_at": "2026-08-16T00:00:00Z",
  "members": [
    {
      "id": "membership-uuid",
      "user_id": "3fa85f64-...",
      "user": {
        "id": "3fa85f64-...",
        "email": "user@example.com",
        "full_name": "John Doe"
      },
      "joined_at": "2026-08-16T00:00:00Z",
      "is_active": true
    }
  ]
}
```

### `GET /groups`

List groups the current user is an **active member** of. **Auth required.**

**Response `200 OK`:** JSON array of group objects with members.

---

### `GET /groups/{group_id}`

Get a single group with members. **Auth required.**

**Errors:**
- `404` — `group not found`

---

### `POST /groups`

Create a group. **Auth required.** The **authenticated user** is the creator and is automatically added as a member (do not send `created_by` — it comes from the token). The group, the creator membership and any extra members are created in one transaction.

**Request Body:**

```json
{
  "name": "Trip to Goa",
  "member_ids": ["3fa85f64-...", "4fa85f64-..."],
  "simplify_debts": true,
  "currency": "INR"
}
```

| Field | Type | Required | Default | Notes |
|-------|------|----------|---------|-------|
| `name` | string | ✅ | — | |
| `member_ids` | UUID[] | ❌ | `[]` | Other members to add. Creator must NOT be included. |
| `simplify_debts` | boolean | ❌ | `true` | Enable debt simplification |
| `currency` | string | ❌ | `"INR"` | Group default currency |

**Response `201 Created`:** Single group object with members.

**Errors:**
- `400` — `name is required`
- `400` — `creator is added automatically and must not be in member_ids`
- `400` — `duplicate member ids provided`
- `404` — `one or more members not found`

---

### `PUT /groups/{group_id}`

Update a group. **Auth required.** All fields optional.

**Request Body:**

```json
{
  "name": "Goa Trip 2026",
  "simplify_debts": false,
  "currency": "USD",
  "is_active": true
}
```

**Response `200 OK`:** Updated group object with members.

**Errors:**
- `400` — `name is required`
- `404` — `group not found`

---

### `DELETE /groups/{group_id}`

Delete a group. **Auth required.** Only the group creator can delete. Memberships, expenses and splits of the group are removed by foreign key cascades.

**Response `200 OK`:** `"Group 'Trip to Goa' deleted successfully!"`

**Errors:**
- `404` — `group not found`
- `403` — `only group creator can delete the group`

---

## Group Members

### `POST /groups/{group_id}/members`

Add a member to a group. **Auth required.** Any authenticated user may add; the added user must exist. Re-adding a previously removed member **reactivates** the original membership (soft delete).

**Request Body:**

```json
{
  "user_id": "3fa85f64-..."
}
```

**Response `200 OK`:** Updated group object with members.

**Errors:**
- `400` — `user_id is required` / `User is already a member of this group`
- `404` — `group not found` / `user not found`

---

### `DELETE /groups/{group_id}/members/{user_id}`

Remove a member from a group. **Auth required.** Only the group creator can remove members; the creator themselves cannot be removed. Removal is a soft delete (`is_active = 0`).

**Response `200 OK`:** `"Member removed successfully!"`

**Errors:**
- `404` — `group not found` / `Member not found in group`
- `403` — `only group creator can remove members`
- `400` — `Cannot remove the group creator`

---

## Expenses

### Split Types

Expenses support four split strategies, specified via the `split_type` field. Splits are computed **server-side** and stored with the expense in one transaction. All amounts are rounded to two decimals; any rounding remainder is assigned to the first split.

| `split_type` | Description | `splits` array required? |
|--------------|-------------|--------------------------|
| `EQUAL` | Split evenly among all **active** group members | ❌ (ignored) |
| `EXACT` | Each split specifies an exact `amount`; the sum must equal the total | ✅ |
| `PERCENTAGE` | Each split specifies a `percentage`; must sum to 100 | ✅ |
| `SHARES` | Each split specifies a `shares` count; amounts are proportional | ✅ |

The `splits` request entries have this shape:

```json
{
  "user_id": "3fa85f64-...",
  "amount": 50.0,
  "percentage": 25.0,
  "shares": 1
}
```

| Field | Type | Used by |
|-------|------|---------|
| `user_id` | UUID | All non-EQUAL types; must be an active group member |
| `amount` | number | `EXACT` |
| `percentage` | number | `PERCENTAGE` |
| `shares` | integer | `SHARES` (must be > 0) |

Expense object shape (with stored splits):

```json
{
  "id": "3fa85f64-...",
  "group_id": "3fa85f64-...",
  "category_id": "3fa85f64-...",
  "paid_by": "3fa85f64-...",
  "amount": 150.0,
  "description": "Dinner",
  "currency": "INR",
  "split_type": "EQUAL",
  "expense_date": "2026-08-16T00:00:00Z",
  "is_active": true,
  "created_at": "2026-08-16T00:00:00Z",
  "updated_at": "2026-08-16T00:00:00Z",
  "splits": [
    {
      "id": "split-uuid",
      "user_id": "3fa85f64-...",
      "amount": 50.0,
      "percentage": null,
      "shares": null
    }
  ]
}
```

### `POST /expenses`

Create an expense. **Auth required.** The `paid_by` field is **always the authenticated user** and must be an active member of the group.

**Request Body (EQUAL split):**

```json
{
  "group_id": "3fa85f64-...",
  "category_id": "3fa85f64-...",
  "amount": 150.0,
  "description": "Dinner",
  "currency": "INR",
  "split_type": "EQUAL",
  "expense_date": "2026-08-16T00:00:00Z",
  "splits": []
}
```

**Request Body (EXACT split):**

```json
{
  "group_id": "3fa85f64-...",
  "category_id": "3fa85f64-...",
  "amount": 150.0,
  "description": "Dinner",
  "expense_date": "2026-08-16T00:00:00Z",
  "split_type": "EXACT",
  "splits": [
    { "user_id": "aaa...", "amount": 100.0 },
    { "user_id": "bbb...", "amount": 50.0 }
  ]
}
```

**Request Body (PERCENTAGE split):**

```json
{
  "group_id": "3fa85f64-...",
  "category_id": "3fa85f64-...",
  "amount": 150.0,
  "description": "Dinner",
  "expense_date": "2026-08-16T00:00:00Z",
  "split_type": "PERCENTAGE",
  "splits": [
    { "user_id": "aaa...", "percentage": 60.0 },
    { "user_id": "bbb...", "percentage": 40.0 }
  ]
}
```

**Request Body (SHARES split):**

```json
{
  "group_id": "3fa85f64-...",
  "category_id": "3fa85f64-...",
  "amount": 150.0,
  "description": "Dinner",
  "expense_date": "2026-08-16T00:00:00Z",
  "split_type": "SHARES",
  "splits": [
    { "user_id": "aaa...", "shares": 2 },
    { "user_id": "bbb...", "shares": 1 }
  ]
}
```

| Field | Type | Required | Default | Notes |
|-------|------|----------|---------|-------|
| `group_id` | UUID | ✅ | — | Group must exist |
| `category_id` | UUID | ✅ | — | Must be a category of that group |
| `amount` | number | ✅ | — | Must be > 0 |
| `description` | string | ✅ | — | |
| `currency` | string | ❌ | `"INR"` | |
| `split_type` | string | ❌ | `"EQUAL"` | `EQUAL`, `EXACT`, `PERCENTAGE`, `SHARES` |
| `expense_date` | datetime | ✅ | — | RFC 3339 |
| `splits` | array | ❌ | `[]` | See split types |

**Response `201 Created`:** Single expense object with computed splits.

**Errors:**
- `400` — `please provide all the required fields [group_id, category_id, amount, description, expense_date]`
- `400` — `expense_date must be a valid RFC3339 datetime`
- `400` — `amount must be greater than zero`
- `400` — `split_type must be one of EQUAL, EXACT, PERCENTAGE, SHARES`
- `400` — `Group has no members`
- `400` — `payer is not an active member of this group` / `split user is not an active member of this group`
- `400` — `splits are required for this split type` / `duplicate user in splits` / `sum of exact splits must equal total amount` / `percentages must sum to 100` / `total shares must be greater than 0`
- `400` — `Category not found`
- `404` — `group not found`

---

### `GET /expenses/group/{group_id}`

List a group's **active** expenses (with splits). **Auth required.** → `200 OK` with a JSON array.

---

### `GET /expenses/{expense_id}`

Get a single expense with splits. **Auth required.**

**Errors:**
- `404` — `expense not found`

---

### `PUT /expenses/{expense_id}`

Update an expense. **Auth required.** All fields optional. Note: `group_id`, `category_id` and `paid_by` are **immutable**, and updating `amount` or `split_type` does **not** recompute the stored splits.

**Request Body:**

```json
{
  "amount": 200.0,
  "description": "Updated dinner",
  "currency": "INR",
  "split_type": "EQUAL",
  "expense_date": "2026-08-16T00:00:00Z",
  "is_active": true
}
```

**Response `200 OK`:** Updated expense object with splits.

**Errors:**
- `400` — `amount must be greater than zero` / `expense_date must be a valid RFC3339 datetime` / `split_type must be one of EQUAL, EXACT, PERCENTAGE, SHARES`
- `404` — `expense not found`

---

### `DELETE /expenses/{expense_id}`

Delete an expense and its splits. **Auth required.**

**Response `200 OK`:** `"Expense deleted successfully!"`

**Errors:**
- `404` — `expense not found`

---

## Balances

Balances are derived from active expenses, their splits, and settlements:

```
net(user) = paid in expenses + paid in settlements
          - owed via splits   - received via settlements
```

All values are rounded to two decimals. `simplified_settlements` is the minimal set of transfers that settles all group debt (greedy max-debtor/max-creditor matching).

### `GET /balances/group/{group_id}`

Get raw balances and simplified settlements for a group. **Auth required.**

**Response `200 OK`:**

```json
{
  "group_id": "3fa85f64-...",
  "group_name": "Trip to Goa",
  "balances": [
    {
      "user_id": "3fa85f64-...",
      "email": "user@example.com",
      "full_name": "John Doe",
      "net_balance": 50.0,
      "status": "owed"
    },
    {
      "user_id": "4fa85f64-...",
      "email": "jane@example.com",
      "full_name": "Jane Doe",
      "net_balance": -50.0,
      "status": "owes"
    }
  ],
  "simplified_settlements": [
    {
      "from_user_id": "4fa85f64-...",
      "to_user_id": "3fa85f64-...",
      "amount": 50.0
    }
  ],
  "total_settlements": 1
}
```

**Balance `status` values:**
- `owed` — positive balance (others owe this user)
- `owes` — negative balance (this user owes others)
- `settled` — zero balance

**Errors:**
- `404` — `group not found`

---

### `GET /balances/me`

Get balances across all groups for the current user. **Auth required.**

**Response `200 OK`:**

```json
{
  "user_id": "3fa85f64-...",
  "total_owed_to_me": 120.0,
  "total_i_owe": 30.0,
  "net_balance": 90.0,
  "groups": [
    {
      "group_id": "3fa85f64-...",
      "group_name": "Trip to Goa",
      "my_net_balance": 50.0,
      "my_settlements": [
        {
          "from_user_id": "4fa85f64-...",
          "to_user_id": "3fa85f64-...",
          "amount": 50.0
        }
      ]
    }
  ]
}
```

---

## Settlements

Settlement object shape:

```json
{
  "id": "3fa85f64-...",
  "group_id": "3fa85f64-...",
  "paid_by": "4fa85f64-...",
  "paid_to": "3fa85f64-...",
  "amount": 50.0,
  "payment_method": "cash",
  "note": "Dinner split",
  "settled_at": "2026-08-16T00:00:00Z",
  "created_at": "2026-08-16T00:00:00Z"
}
```

### `GET /settlements/group/{group_id}`

List a group's settlements, most recent first. **Auth required.** → `200 OK` with a JSON array.

---

### `POST /settlements`

Record a settlement. **Auth required.** The `paid_by` field is **always the authenticated user**; both payer and recipient must be active members of the group.

**Request Body:**

```json
{
  "group_id": "3fa85f64-...",
  "paid_to": "3fa85f64-...",
  "amount": 50.0,
  "payment_method": "cash",
  "note": "Dinner split"
}
```

| Field | Type | Required | Default | Notes |
|-------|------|----------|---------|-------|
| `group_id` | UUID | ✅ | — | |
| `paid_to` | UUID | ✅ | — | Recipient; must be an active group member |
| `amount` | number | ✅ | — | Must be > 0 |
| `payment_method` | string | ❌ | `"cash"` | `cash`, `bank` or `upi` |
| `note` | string | ❌ | `""` | |

**Response `201 Created`:** Single settlement object.

**Errors:**
- `400` — `cannot settle with yourself`
- `400` — `recipient is not a member of this group` / `payer is not a member of this group`
- `400` — `settlement amount must be positive` / `paid_to is required` / `group_id is required`
- `400` — `payment_method must be one of cash, bank, upi`
- `404` — `group not found`

---

### `DELETE /settlements/{settlement_id}`

Delete a settlement. **Auth required.** Only the payer can delete.

**Response `200 OK`:** `"Settlement deleted successfully!"`

**Errors:**
- `404` — `settlement not found`
- `403` — `Only the payer can delete a settlement`

---

## Data Types & Conventions

| Type | JSON Representation | Notes |
|------|---------------------|-------|
| UUID | string | e.g. `"3fa85f64-5717-4562-b3fc-2c963f66afa6"` |
| money | number | e.g. `150.5` — always rounded to two decimals |
| datetime | string | RFC 3339, e.g. `"2026-08-16T00:00:00Z"` |
| boolean | boolean | `true` / `false` |
| nullable | `null` | e.g. `percentage`, `shares`, `updated_at` |

---

## Error Responses

Errors are returned as JSON with an `error` field:

```json
{
  "error": "Error message here"
}
```

### Common Status Codes

| Status | Meaning |
|--------|---------|
| `200` | Success |
| `201` | Resource created |
| `400` | Bad request / validation error |
| `401` | Unauthorized (missing/invalid/expired token) |
| `403` | Forbidden (insufficient permissions) |
| `404` | Not found |
| `500` | Internal server error |

### Authentication Errors

- `401` — `Invalid token` (missing or malformed header, or bad signature)
- `401` — `invalid or expired token`
- `401` — `invalid token type` (e.g. using a refresh token as access token or vice versa)
- `403` — `account is deactivated` (login only)

---

## Quick Reference — Endpoint Summary

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| POST | `/auth/register` | ❌ | Register user |
| POST | `/auth/login` | ❌ | Login, get tokens |
| POST | `/auth/refresh` | ❌ | Refresh tokens |
| GET | `/auth/me` | ✅ | Current user |
| GET | `/users` | ✅ | List users |
| GET | `/users/{id}` | ✅ | Get user |
| POST | `/users` | ❌ | Create user |
| PUT | `/users/{id}` | ✅ | Update user |
| DELETE | `/users/{id}` | ✅ | Delete user |
| POST | `/categories` | ✅ | Create category (group-scoped) |
| GET | `/categories/{group_id}` | ✅ | List group categories |
| GET | `/categories/{group_id}/{id}` | ✅ | Get category |
| PUT | `/categories/{group_id}/{id}` | ✅ | Update category |
| DELETE | `/categories/{group_id}/{id}` | ✅ | Delete category |
| GET | `/groups` | ✅ | List my groups |
| GET | `/groups/{id}` | ✅ | Get group |
| POST | `/groups` | ✅ | Create group |
| PUT | `/groups/{id}` | ✅ | Update group |
| DELETE | `/groups/{id}` | ✅ | Delete group |
| POST | `/groups/{group_id}/members` | ✅ | Add member |
| DELETE | `/groups/{group_id}/members/{user_id}` | ✅ | Remove member |
| POST | `/expenses` | ✅ | Create expense (splits computed) |
| GET | `/expenses/group/{group_id}` | ✅ | List group expenses |
| GET | `/expenses/{id}` | ✅ | Get expense |
| PUT | `/expenses/{id}` | ✅ | Update expense |
| DELETE | `/expenses/{id}` | ✅ | Delete expense |
| GET | `/balances/group/{group_id}` | ✅ | Group balances |
| GET | `/balances/me` | ✅ | My balances |
| POST | `/settlements` | ✅ | Record settlement |
| GET | `/settlements/group/{group_id}` | ✅ | List settlements |
| DELETE | `/settlements/{id}` | ✅ | Delete settlement |
