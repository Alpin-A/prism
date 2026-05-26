
import argparse
import random
import time
import urllib.request
import urllib.error
import json

BASE_URL = "http://localhost:8080"

CONTROL_RATE    = 0.10   # 10% conversion rate for control
TREATMENT_RATE  = 0.17   # 17% conversion rate for treatment — detectable lift


def post(path: str, body: dict) -> dict | None:
    data = json.dumps(body).encode()
    req = urllib.request.Request(
        f"{BASE_URL}{path}",
        data=data,
        headers={"Content-Type": "application/json"},
        method="POST",
    )
    try:
        with urllib.request.urlopen(req, timeout=5) as resp:
            body = resp.read()
            return json.loads(body) if body else {}
    except urllib.error.HTTPError as e:
        if e.code == 409:
            return None  # already exists, fine
        print(f"POST {path} failed: {e.code} {e.reason}")
        return None
    except Exception as e:
        print(f"POST {path} error: {e}")
        return None


def get(path: str) -> dict | None:
    try:
        with urllib.request.urlopen(f"{BASE_URL}{path}", timeout=5) as resp:
            return json.loads(resp.read())
    except Exception as e:
        print(f"GET {path} error: {e}")
        return None


def ensure_experiment(experiment_id: str) -> bool:
    # Try to fetch first — if it exists, use it.
    existing = get(f"/api/v1/experiments/{experiment_id}")
    if existing and "id" in existing:
        print(f"Using existing experiment: {experiment_id}")
        return True

    result = post("/api/v1/experiments", {
        "id":          experiment_id,
        "name":        "Demo: Homepage CTA",
        "description": "Traffic generator demo experiment",
        "metric_type": "conversion",
        "variants": [
            {"id": "control",   "name": "Original", "weight": 0.5},
            {"id": "treatment", "name": "Variant",  "weight": 0.5},
        ],
    })
    if result and "id" in result:
        print(f"Created experiment: {experiment_id}")
        return True

    print("Failed to create experiment")
    return False


def simulate_user(experiment_id: str, user_id: str, rng: random.Random) -> None:
    # Assign the user to a variant.
    resp = get(f"/api/v1/assign?experiment_id={experiment_id}&user_id={user_id}")
    if not resp or "variant_id" not in resp:
        return

    variant_id = resp["variant_id"]

    # Decide whether this user converts based on their variant's rate.
    conversion_rate = TREATMENT_RATE if variant_id == "treatment" else CONTROL_RATE
    if rng.random() < conversion_rate:
        post("/api/v1/events", {
            "experiment_id": experiment_id,
            "user_id":       user_id,
            "variant_id":    variant_id,
            "event_type":    "conversion",
            "value":         1.0,
        })


def run(experiment_id: str, rate: int, duration: int) -> None:
    if not ensure_experiment(experiment_id):
        return

    rng = random.Random(42)
    user_counter = 0
    start = time.time()
    interval = 1.0 / rate

    print(f"\nSimulating {rate} users/sec for {duration}s")
    print(f"Control conversion rate:   {CONTROL_RATE*100:.0f}%")
    print(f"Treatment conversion rate: {TREATMENT_RATE*100:.0f}%")
    print(f"Expected lift: {(TREATMENT_RATE-CONTROL_RATE)/CONTROL_RATE*100:.0f}% relative\n")
    print("Running... (Ctrl+C to stop early)\n")

    try:
        while time.time() - start < duration:
            tick = time.time()
            user_id = f"demo_user_{user_counter}"
            simulate_user(experiment_id, user_id, rng)
            user_counter += 1

            elapsed = time.time() - tick
            sleep_time = interval - elapsed
            if sleep_time > 0:
                time.sleep(sleep_time)

            if user_counter % (rate * 10) == 0:
                seconds_elapsed = int(time.time() - start)
                print(f"  {seconds_elapsed}s elapsed — {user_counter} users simulated")

    except KeyboardInterrupt:
        pass

    total_time = time.time() - start
    print(f"\nDone. {user_counter} users simulated in {total_time:.1f}s")
    print(f"Check results: curl -s 'http://localhost:8080/api/v1/experiments/{experiment_id}/results?event_type=conversion' | python3 -m json.tool")


if __name__ == "__main__":
    parser = argparse.ArgumentParser(description="Prism demo traffic generator")
    parser.add_argument("--experiment", default="demo_homepage_cta",
                        help="Experiment ID (default: demo_homepage_cta)")
    parser.add_argument("--rate", type=int, default=20,
                        help="Users per second (default: 20)")
    parser.add_argument("--duration", type=int, default=180,
                        help="Duration in seconds (default: 180)")
    args = parser.parse_args()

    run(args.experiment, args.rate, args.duration)