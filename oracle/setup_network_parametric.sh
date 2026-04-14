#!/bin/sh
set -eu

echo "[NETWORK] Parametric tc initialization for ORACLE_ID=${ORACLE_ID:-unset}"

if [ "${ENABLE_LATENCY:-true}" != "true" ]; then
    echo "[NETWORK] Latency disabled; skipping tc setup"
    exec /usr/local/bin/wait-for-deploy.sh "$@"
fi

: "${ORACLE_ID:?ORACLE_ID is required}"
: "${NUM_ORACLES:?NUM_ORACLES is required}"
: "${NETWORK_SEED:?NETWORK_SEED is required}"
: "${ORACLE_IPS:?ORACLE_IPS is required}"

NETWORK_LOCATIONS="${NETWORK_LOCATIONS:-Milan,Toronto,Moscow,Lisbon,Mumbai,Johannesburg,NewYork}"

is_uint() {
    case "$1" in
        ''|*[!0-9]*) return 1 ;;
        *) return 0 ;;
    esac
}

csv_count() {
    printf '%s\n' "$1" | awk -F, '{ print NF }'
}

csv_field() {
    printf '%s\n' "$1" | awk -F, -v n="$2" '{
        value = $n
        gsub(/^[[:space:]]+|[[:space:]]+$/, "", value)
        print value
    }'
}

canonical_location() {
    key="$(printf '%s' "$1" | tr '[:upper:]' '[:lower:]' | tr -d ' _-\r')"
    case "$key" in
        milan|milano) echo "Milan" ;;
        newyork|ny) echo "NewYork" ;;
        mumbai) echo "Mumbai" ;;
        johannesburg|joburg) echo "Johannesburg" ;;
        lisbon|lisboa) echo "Lisbon" ;;
        moscow|mosca) echo "Moscow" ;;
        toronto) echo "Toronto" ;;
        *) echo "$1" ;;
    esac
}

choose_location() {
    oracle_index="$1"
    location_count="$(csv_count "$NETWORK_LOCATIONS")"
    if [ "$location_count" -lt 1 ]; then
        echo "[NETWORK] NETWORK_LOCATIONS must contain at least one location" >&2
        exit 1
    fi

    hex="$(printf '%s:%s' "$NETWORK_SEED" "$oracle_index" | sha256sum | awk '{ print substr($1, 1, 6) }')"
    value=$((0x$hex))
    field=$((value % location_count + 1))
    canonical_location "$(csv_field "$NETWORK_LOCATIONS" "$field")"
}

latency_between() {
    src="$(canonical_location "$1")"
    dst="$(canonical_location "$2")"

    if [ "$src" = "$dst" ]; then
        echo 0
        return
    fi

    case "$src:$dst" in
        Milan:NewYork) echo 90 ;;
        Milan:Mumbai) echo 120 ;;
        Milan:Johannesburg) echo 175 ;;
        Milan:Lisbon) echo 50 ;;
        Milan:Moscow) echo 45 ;;
        Milan:Toronto) echo 110 ;;

        NewYork:Milan) echo 90 ;;
        NewYork:Mumbai) echo 190 ;;
        NewYork:Johannesburg) echo 230 ;;
        NewYork:Lisbon) echo 115 ;;
        NewYork:Moscow) echo 120 ;;
        NewYork:Toronto) echo 17 ;;

        Mumbai:Milan) echo 120 ;;
        Mumbai:NewYork) echo 190 ;;
        Mumbai:Johannesburg) echo 290 ;;
        Mumbai:Lisbon) echo 150 ;;
        Mumbai:Moscow) echo 175 ;;
        Mumbai:Toronto) echo 240 ;;

        Johannesburg:Milan) echo 175 ;;
        Johannesburg:NewYork) echo 230 ;;
        Johannesburg:Mumbai) echo 290 ;;
        Johannesburg:Lisbon) echo 210 ;;
        Johannesburg:Moscow) echo 200 ;;
        Johannesburg:Toronto) echo 225 ;;

        Lisbon:Milan) echo 50 ;;
        Lisbon:NewYork) echo 110 ;;
        Lisbon:Mumbai) echo 150 ;;
        Lisbon:Johannesburg) echo 210 ;;
        Lisbon:Moscow) echo 80 ;;
        Lisbon:Toronto) echo 120 ;;

        Moscow:Milan) echo 45 ;;
        Moscow:NewYork) echo 120 ;;
        Moscow:Mumbai) echo 175 ;;
        Moscow:Johannesburg) echo 200 ;;
        Moscow:Lisbon) echo 80 ;;
        Moscow:Toronto) echo 140 ;;

        Toronto:Milan) echo 110 ;;
        Toronto:NewYork) echo 17 ;;
        Toronto:Mumbai) echo 240 ;;
        Toronto:Johannesburg) echo 230 ;;
        Toronto:Lisbon) echo 120 ;;
        Toronto:Moscow) echo 140 ;;

        *)
            echo "[NETWORK] Missing latency pair: $src -> $dst" >&2
            exit 1
            ;;
    esac
}

if ! is_uint "$ORACLE_ID" || ! is_uint "$NUM_ORACLES"; then
    echo "[NETWORK] ORACLE_ID and NUM_ORACLES must be non-negative integers" >&2
    exit 1
fi

if [ "$ORACLE_ID" -ge "$NUM_ORACLES" ]; then
    echo "[NETWORK] ORACLE_ID=$ORACLE_ID is outside NUM_ORACLES=$NUM_ORACLES" >&2
    exit 1
fi

ip_count="$(csv_count "$ORACLE_IPS")"
if [ "$ip_count" -lt "$NUM_ORACLES" ]; then
    echo "[NETWORK] ORACLE_IPS has $ip_count entries but NUM_ORACLES=$NUM_ORACLES" >&2
    exit 1
fi

src_location="$(choose_location "$ORACLE_ID")"
echo "[NETWORK] oracle${ORACLE_ID} assigned location: ${src_location}"

tc qdisc del dev eth0 root 2>/dev/null || true
tc qdisc add dev eth0 root handle 1: prio bands "$((NUM_ORACLES + 1))"

i=0
band=1
while [ "$i" -lt "$NUM_ORACLES" ]; do
    if [ "$i" != "$ORACLE_ID" ]; then
        dst_ip="$(csv_field "$ORACLE_IPS" "$((i + 1))")"
        dst_location="$(choose_location "$i")"
        delay_ms="$(latency_between "$src_location" "$dst_location")"

        if [ -n "$dst_ip" ] && [ "$delay_ms" != "0" ]; then
            band="$((band + 1))"
            handle="$((band * 10))"

            echo "[NETWORK] oracle${ORACLE_ID} (${src_location}) -> oracle${i} (${dst_location}) ${dst_ip}: ${delay_ms}ms"

            tc qdisc add dev eth0 parent "1:${band}" handle "${handle}:" netem delay "${delay_ms}ms"
            tc filter add dev eth0 protocol ip parent 1:0 prio "$band" \
                u32 match ip dst "$dst_ip" flowid "1:${band}"
        fi
    fi

    i="$((i + 1))"
done

echo "[NETWORK] Parametric tc rules applied successfully"
exec /usr/local/bin/wait-for-deploy.sh "$@"
