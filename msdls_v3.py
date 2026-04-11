import argparse
import json
import logging
import uuid
import time
import requests

# New v3 JSON Endpoints
BASE_URL = "https://www.microsoft.com/software-download-connector/api"
PROFILE = "606624d44113"
LOCALE = "en-US"
UA = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36"

def setup_session():
    """Initializes a session with Microsoft tracking servers."""
    session_id = str(uuid.uuid4())
    return session_id

def get_product(product_id, session_id):
    """Fetches product/SKU info for a given ID."""
    url = f"{BASE_URL}/getskuinformationbyproductedition"
    params = {
        "profile": PROFILE,
        "productEditionId": product_id,
        "SKU": "undefined",
        "friendlyFileName": "undefined",
        "Locale": LOCALE,
        "sessionID": session_id
    }
    headers = {
        "User-Agent": UA, 
        "Accept": "application/json",
        "Referer": "https://www.microsoft.com/en-us/software-download/windows11"
    }
    
    try:
        r = requests.get(url, params=params, headers=headers, timeout=10)
        if not r.ok: return None
        
        # Handle Microsoft's double-encoded JSON
        raw_text = r.text
        try:
            # First pass: decode the wrapper JSON
            data = json.loads(raw_text)
            # Second pass: if the value is a string, it's the real JSON
            if isinstance(data, str):
                data = json.loads(data)
            return data
        except:
            return None
    except Exception as e:
        logging.error(f"Error fetching ID {product_id}: {e}")
        return None

def scan_id(product_id):
    """Checks if a product ID is valid and active."""
    session_id = setup_session()
    data = get_product(product_id, session_id)
    
    if not data or "Skus" not in data or len(data["Skus"]) == 0:
        return None
    
    if "Errors" in data and data["Errors"] and len(data["Errors"]) > 0:
        return None

    # MS usually puts the release name in the first SKU
    skus = data.get("Skus", [])
    if skus:
        # Check multiple possible name keys
        s = skus[0]
        # ReleaseName is often "Windows 10 Version 22H2 (Updated Oct 2025)"
        name = s.get("EditionName") or s.get("ReleaseName") or s.get("FriendlyName")
        if name:
            return name
            
    return f"Windows Product {product_id}"

if __name__ == "__main__":
    parser = argparse.ArgumentParser(description="Scan for new Windows Product IDs (v3 JSON API)")
    parser.add_argument("--first", required=True, type=int, help="First ID to check")
    parser.add_argument("--last", required=True, type=int, help="Last ID to check")
    parser.add_argument("--write", help="Output JSON file")
    args = parser.parse_args()

    logging.basicConfig(level=logging.INFO, format='%(message)s')
    products = {}

    logging.info(f"Scanning range {args.first} to {args.last}...")

    for i in range(args.first, args.last + 1):
        name = scan_id(i)
        if name:
            logging.info(f"✅ FOUND: [{i}] {name}")
            products[str(i)] = name
        else:
            # Only log skips for very small ranges to keep output clean
            if (args.last - args.first) < 20:
                logging.info(f"❌ SKIP:  [{i}] Not found")
        
        # Be gentle to the API
        time.sleep(0.5)

    if args.write:
        import os
        catalog = {}
        if os.path.exists(args.write):
            try:
                with open(args.write, 'r', encoding='utf-8') as f:
                    catalog = json.load(f)
            except Exception as e:
                logging.error(f"Failed to read existing catalog: {e}")
        
        # Merge new products, preserving existing extra metadata if present
        for pid, name in products.items():
            if pid in catalog:
                # Update name but keep other fields like badge, archs, etc.
                if isinstance(catalog[pid], dict):
                    catalog[pid]["name"] = name
                else:
                    # Upgrade from old flat format
                    catalog[pid] = {
                        "name": name,
                        "badge": "",
                        "archs": [],
                        "related": []
                    }
            else:
                catalog[pid] = {
                    "name": name,
                    "badge": "",
                    "archs": [],
                    "related": []
                }
                
        with open(args.write, 'w', encoding='utf-8') as f:
            json.dump(catalog, f, indent=4)
            logging.info(f"\nSaved {len(catalog)} products to {args.write} (merged with existing)")
    
    logging.info("Done.")
