#!/usr/bin/env bats

load helpers.sh

# JWTs generated at: https://jwt.io/
# Algorithm RS256
# Payload:
# {
#   "exp": from https://www.unixtimestamp.com/,
#   "action": "READ"
# }
# RSA keys used: tests/privkey.pem and tests/pubkey.crt

@test "checking JWT missing header" {
  run GET "$CACHE/get"
  [[ "$status" -eq 0 ]]
  [[ "$output" == "401" ]]
}

@test "checking JWT expired" {
  run GET "$CACHE/get" \
    -H "Authorization: Bearer eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9.eyJleHAiOjE2MDc4NzIyMDYsImFjdGlvbiI6IlJFQUQifQ.aNeWCA7yMdqEoeSFEb8rRAjntz1rMvBqvaRYPEvs_KwWohr_unj-EceMkX0_31otWdN2UtmH9mnPavebTah2B3xemtZpd5RviKu-NL_NXop5rTzOUYUI3pRHviok6bxYwx-nDt23p2w0VpfUiHtfLmvWa0XKvoONq7g6Iq4Y1P33JvVV2XgdYbKhLOlPPsuWdTFADXPLb8yPYzm0Blyz2LLw0pKlmLccx4h0sw24CweuKAz7r7HCoQc14vka3XmEtDwmkuTqoXJT9zJ8-DOSjVWMZERz827msHwwYF46Ge2KrSNa3qtdTSUCKdW_KpGo9SFJCrPRfJ4Fh5h7kqBmFQ"
  [[ "$status" -eq 0 ]]
  [[ "$output" == "401" ]]
}

@test "checking JWT malformed segments" {
  run GET "$CACHE/get" \
    -H "Authorization: Bearer eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9.eyJleHAiOjE2MDc4NzIyMDYsImFjdGlvbiI6IlJFQUQifQ"
  [[ "$status" -eq 0 ]]
  [[ "$output" == "401" ]]
}

@test "checking JWT bad signature" {
  run GET "$CACHE/get" \
    -H "Authorization: Bearer eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9.eyJleHAiOjQxMzIzOTM4MDYsImFjdGlvbiI6IlJFQUQifQ.asdf"
  [[ "$status" -eq 0 ]]
  [[ "$output" == "401" ]]
}

@test "checking JWT action not allowed" {
  # POST when only action allowed is READ
  run POST "$CACHE/upload" \
    -H "Authorization: Bearer eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9.eyJleHAiOjQxMzIzOTM4MDYsImFjdGlvbiI6IlJFQUQifQ.J37qq96xRwYZD4Qg01_Xm999PRhNcmvxYc4gZEoPjxZxQTZh4AwG6x3KIdo01wWGbkKj0YR2QB-UR7IvPPeC9vomWFPB-No2G0sDY8EYMEt5ZgJbpJg6uzQJO1SMGaYCFc57TbP1KVOtSoavf4NLYzT4n3XgiDV-wZn4XSXtHBlbiuxXkmbVWRjOlRUdEmKIHyyYwkXHL7hiDGJRPNLSn4wBnFyrVrviKgxcauuM6pJ9SXFLc-yd9HKrwig_9K4ZthIeldXd9aRjcv37IuIGqzvWR4KBew9XGbg2ZXMMKbzr1rba1fWlrRlxU3KsRPL5kmnYhJOG3et_m37Wcm9agg"
  [[ "$status" -eq 0 ]]
  [[ "$output" == "401" ]]
}

@test "checking JWT correct" {
  run GET "$CACHE/get?bucket=test&key=test" \
    -H "Authorization: Bearer eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9.eyJleHAiOjQxMzIzOTM4MDYsImFjdGlvbiI6IlJFQUQifQ.J37qq96xRwYZD4Qg01_Xm999PRhNcmvxYc4gZEoPjxZxQTZh4AwG6x3KIdo01wWGbkKj0YR2QB-UR7IvPPeC9vomWFPB-No2G0sDY8EYMEt5ZgJbpJg6uzQJO1SMGaYCFc57TbP1KVOtSoavf4NLYzT4n3XgiDV-wZn4XSXtHBlbiuxXkmbVWRjOlRUdEmKIHyyYwkXHL7hiDGJRPNLSn4wBnFyrVrviKgxcauuM6pJ9SXFLc-yd9HKrwig_9K4ZthIeldXd9aRjcv37IuIGqzvWR4KBew9XGbg2ZXMMKbzr1rba1fWlrRlxU3KsRPL5kmnYhJOG3et_m37Wcm9agg"
  [[ "$status" -eq 0 ]]
  [[ "$output" == "404" ]]
}