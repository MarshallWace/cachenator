#!/usr/bin/env bats

load helpers.sh

# JWTs generated at: https://jwt.io/
# Algorithm RS256
# Payload:
# {
#   "exp": from https://www.unixtimestamp.com/,
#   "aud": "cachenator",
#   "iss": "auth-provider",
#   "action": "READ"
# }
# Not actually sensitive keys, only used for local testing for JWT tests in here
# RSA keys used: tests/privkey.pem and tests/pubkey.crt

@test "missing auth header" {
  run GET "$CACHE/get"
  [[ "$status" -eq 0 ]]
  [[ "$output" == "401" ]]
}

@test "JWT expired" {
  run GET "$CACHE/get" \
    -H "Authorization: Bearer eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9.eyJleHAiOjE2MDgwNDEwNjAsImF1ZCI6ImNhY2hlbmF0b3IiLCJpc3MiOiJhdXRoLXByb3ZpZGVyIiwiYWN0aW9uIjoiUkVBRCJ9.weH0WM237Y1hJfrCiT0R0OBfpLfyhVxmOz_vgKobXtq6JMOMLNbS08szuLC9hJtTcLPKZ7kXOQ5OGg9U5qXVk2audqGcbYB71FRqrjKwqjLQ_cJIFeh2CsluYrw2iL04z2DUm8gUJphiErQok9H6QbaTzxVbAAwFt1aFud9tMxOK8XY4CnSqYKzHZhJtjgaJhJFVQTuLAfU4tm0652PPrW8eHK2-pqymvuzJFLgr_7EBeaMpX5Ir1UruARw5Y79IeTsyJ5TXUQc5jMCKNu2zA41usn72DNRbihlkUJr1I4cp_Zj0bV9oRAv25ST7g_13SBbz5MHgonbiiFxy_rl4Rg"
  [[ "$status" -eq 0 ]]
  [[ "$output" == "401" ]]
}

@test "JWT malformed segments" {
  run GET "$CACHE/get" \
    -H "Authorization: Bearer eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9.eyJleHAiOjQxMzI1NjI2NjAsImF1ZCI6ImNhY2hlbmF0b3IiLCJpc3MiOiJhdXRoLXByb3ZpZGVyIiwiYWN0aW9uIjoiUkVBRCJ9"
  [[ "$status" -eq 0 ]]
  [[ "$output" == "401" ]]
}

@test "JWT bad signature" {
  run GET "$CACHE/get" \
    -H "Authorization: Bearer eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9.eyJleHAiOjQxMzI1NjI2NjAsImF1ZCI6ImNhY2hlbmF0b3IiLCJpc3MiOiJhdXRoLXByb3ZpZGVyIiwiYWN0aW9uIjoiUkVBRCJ9.asdf"
  [[ "$status" -eq 0 ]]
  [[ "$output" == "401" ]]
}

@test "JWT bad issuer" {
  run GET "$CACHE/get" \
    -H "Authorization: Bearer eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9.eyJleHAiOjQxMzI1NjI2NjAsImF1ZCI6ImNhY2hlbmF0b3IiLCJpc3MiOiJhc2RmIiwiYWN0aW9uIjoiUkVBRCJ9.B0x2OF-Rvu6WpAlGxB4rTJo1CnDN1Hw3gyMah8_ICqWazFnODBxVuENHp7Ky7JSFiMLdb3rYO5R_EdjkFp8totx-PFFwDcIqE0xJ1jJZQ2_GA2ghKg94-KWtcrToSiq3OZcmb0Tr7vtJZvTS92mwH5YQKUDccKOjp8r_cGOMRpZ7jV4o7VlJBtwxVPbFux_oGW7n44eJ0N4AdYQpAkARGfIuyH-UTusreLsfr-xOKSUc6UZ9ZJNGPGX8wj8XRMAn834SnGD-3cwzEEfxMwmK_H8BrMf0Aqj8GAx-ohM8M1T30-Z1uDdurEjGPD7nKKpQzbmS5qrlB4qH5zt8t7EFnA"
  [[ "$status" -eq 0 ]]
  [[ "$output" == "401" ]]
}

@test "JWT bad audience" {
  run GET "$CACHE/get" \
    -H "Authorization: Bearer eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9.eyJleHAiOjQxMzI1NjI2NjAsImF1ZCI6ImFzZGYiLCJpc3MiOiJhdXRoLXByb3ZpZGVyIiwiYWN0aW9uIjoiUkVBRCJ9.twGDU2LvwV-WXZusOrRO6ZqPNvDE9xvblW-qxMG3RZtxmkPALJhipiIGrSlnGgNrjZ_tlKmYl2bWSSC7r4qGITSBzEAmhaljUZQkXBDSnMZwMrppzjal5JkVLaelnQA9sqoKMLcAfdW5L7jrK4BMXQOjSmwF4bKtDOczKbP2WvM6QbWR_mpuNGPnb4xr625f3DoYEiUaA8vknMyUfqufPfpmieQPXFTaUIzPuMOYNUqDvJWR8697LNmWpHUL0oUA4Yvs863VWvqBda4rV2k5VShffARt3j1N6eqV_JRKrDG0uCDREY56E2YJlaneJiaSqXRlFl_HcO5NkWfHh4yJXw"
  [[ "$status" -eq 0 ]]
  [[ "$output" == "401" ]]
}

@test "JWT action not allowed" {
  # POST when only action allowed is READ
  run POST "$CACHE/upload" \
    -H "Authorization: Bearer eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9.eyJleHAiOjQxMzI1NjI2NjAsImF1ZCI6ImNhY2hlbmF0b3IiLCJpc3MiOiJhdXRoLXByb3ZpZGVyIiwiYWN0aW9uIjoiUkVBRCJ9.1SYAXUq0zYwT8DzzfMr-0oE4LO_gCX9kFXb4Ew-95mbujMPgjBnlhqwv8DZdrps72MDQaXbLakCGvZ_HNC55LMV_gG3Q3Nyg4PRV9BP6_6tZPgRbjPOFdq0HpCSWD1w2NjidqbfW4vuB5WAs0Mf1J9g6r-Dzvijjn81YcQznqWb43YocCxWoNWgTyyZwPXTWpmJdgrt9kfcDzB-Z71ezKt6jstUhk_ie7rjfh1viECyfeDkH7OF0qOovGyF7z09ZFrXLJXGQon8phSO4d5J_x8lyLpKagkxInGFhIs2q_aAl_DcKxS1G53HonMZ0DGNj6mZniQVHK5AdJ0QJt7eDYQ"
  [[ "$status" -eq 0 ]]
  [[ "$output" == "401" ]]
}

@test "JWT correct" {
  run GET "$CACHE/get?bucket=test&key=asdf" \
    -H "Authorization: Bearer eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9.eyJleHAiOjQxMzI1NjI2NjAsImF1ZCI6ImNhY2hlbmF0b3IiLCJpc3MiOiJhdXRoLXByb3ZpZGVyIiwiYWN0aW9uIjoiUkVBRCIsImJ1Y2tldCI6InRlc3QiLCJwcmVmaXgiOiJhc2RmIn0.pzAVlCDgZ4BmYZppcjaaKftoV5j0Fm5OhmUcbAJ4oGvlKeq1D4Pgm9xkB0afiBWfOmNLV5S5Ib8bVz0qiIholAeyTtQzNWc5pPe8H20_KMMAzuM-si3G_NdTFsWH-xR_WXBAlYaRR96NruT2mPe333LkEsbhnJHBP8uXQikRK7t4WBrafv7OEnoOYa2DUQpNoqqTmKj9t4SS9zUySV6dD9xUUYo2JOZZ8JdemXPe8C1OFJz__ibxcmpCWMhIJQYFEexa2aKvQEtD2zLXusOmLiN2dTKU9q_yky1hncb5yaVrQUaHulusl_u0CPazi7XOkO8MpLDOw8ounFOWIyK7XQ"
  [[ "$status" -eq 0 ]]
  [[ "$output" == "404" ]]
}

@test "JWT bucket invalid test" {
  run GET "$CACHE/get?bucket=testfail&key=asdf" \
    -H "Authorization: Bearer eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9.eyJleHAiOjQxMzI1NjI2NjAsImF1ZCI6ImNhY2hlbmF0b3IiLCJpc3MiOiJhdXRoLXByb3ZpZGVyIiwiYWN0aW9uIjoiUkVBRCIsImJ1Y2tldCI6InRlc3QiLCJwcmVmaXgiOiJhc2RmIn0.pzAVlCDgZ4BmYZppcjaaKftoV5j0Fm5OhmUcbAJ4oGvlKeq1D4Pgm9xkB0afiBWfOmNLV5S5Ib8bVz0qiIholAeyTtQzNWc5pPe8H20_KMMAzuM-si3G_NdTFsWH-xR_WXBAlYaRR96NruT2mPe333LkEsbhnJHBP8uXQikRK7t4WBrafv7OEnoOYa2DUQpNoqqTmKj9t4SS9zUySV6dD9xUUYo2JOZZ8JdemXPe8C1OFJz__ibxcmpCWMhIJQYFEexa2aKvQEtD2zLXusOmLiN2dTKU9q_yky1hncb5yaVrQUaHulusl_u0CPazi7XOkO8MpLDOw8ounFOWIyK7XQ"
  [[ "$status" -eq 0 ]]
  [[ "$output" == "401" ]]
}

@test "JWT prefix invalid test" {
  run GET "$CACHE/get?bucket=test&key=keyfail" \
    -H "Authorization: Bearer eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9.eyJleHAiOjQxMzI1NjI2NjAsImF1ZCI6ImNhY2hlbmF0b3IiLCJpc3MiOiJhdXRoLXByb3ZpZGVyIiwiYWN0aW9uIjoiUkVBRCIsImJ1Y2tldCI6InRlc3QiLCJwcmVmaXgiOiJhc2RmIn0.pzAVlCDgZ4BmYZppcjaaKftoV5j0Fm5OhmUcbAJ4oGvlKeq1D4Pgm9xkB0afiBWfOmNLV5S5Ib8bVz0qiIholAeyTtQzNWc5pPe8H20_KMMAzuM-si3G_NdTFsWH-xR_WXBAlYaRR96NruT2mPe333LkEsbhnJHBP8uXQikRK7t4WBrafv7OEnoOYa2DUQpNoqqTmKj9t4SS9zUySV6dD9xUUYo2JOZZ8JdemXPe8C1OFJz__ibxcmpCWMhIJQYFEexa2aKvQEtD2zLXusOmLiN2dTKU9q_yky1hncb5yaVrQUaHulusl_u0CPazi7XOkO8MpLDOw8ounFOWIyK7XQ"
  [[ "$status" -eq 0 ]]
  [[ "$output" == "401" ]]
}
