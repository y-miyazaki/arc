#!/usr/bin/env bats

# Tests for scripts/lib/csv.sh

setup() {
    source "scripts/lib/csv.sh"
}

@test "normalize_csv_value returns empty string for empty/null" {
    run normalize_csv_value ""
    [ "$status" -eq 0 ]
    [ "$output" = "" ]

    run normalize_csv_value "null"
    [ "$status" -eq 0 ]
    [ "$output" = "" ]
}

@test "normalize_csv_value does not quote simple string" {
    run normalize_csv_value "simple"
    [ "$status" -eq 0 ]
    [ "$output" = "simple" ]
}

@test "normalize_csv_value quotes string with comma" {
    run normalize_csv_value "foo,bar"
    [ "$status" -eq 0 ]
    [ "$output" = '"foo,bar"' ]
}

@test "normalize_csv_value quotes string with quote" {
    run normalize_csv_value 'foo"bar'
    [ "$status" -eq 0 ]
    [ "$output" = '"foo""bar"' ]
}

@test "normalize_csv_value quotes string with newline" {
    run normalize_csv_value $'foo\nbar'
    [ "$status" -eq 0 ]
    # Should be quoted because it contains newline
    [[ "$output" == "\"foo"* ]]
    [[ "$output" == *"bar\"" ]]
}
