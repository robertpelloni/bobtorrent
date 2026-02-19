from playwright.sync_api import sync_playwright

def verify_advanced_ingest_ui():
    with sync_playwright() as p:
        browser = p.chromium.launch()
        page = browser.new_page()
        try:
            print("Navigating to http://localhost:3000...")
            page.goto("http://localhost:3000")

            # Wait for initial load
            page.wait_for_selector("#dashboard")

            print("Clicking Publish tab...")
            page.click("button[data-tab='publish']")

            # Find Advanced Toggle
            print("Checking Advanced Options...")
            page.check("#ingest-adv-toggle")

            # Verify Strategy Dropdown visible
            page.wait_for_selector("#ingest-advanced:not(.hidden)")

            # Select Erasure Coding
            print("Selecting Erasure Coding...")
            page.select_option("#ingest-strategy", "erasure")

            # Verify Shard inputs visible
            page.wait_for_selector("#ingest-ec-settings:not(.hidden)")

            # Fill inputs
            page.fill("#ingest-data-shards", "6")
            page.fill("#ingest-parity-shards", "3")

            # Take screenshot
            screenshot_path = "verification/advanced_ingest.png"
            page.screenshot(path=screenshot_path)
            print(f"Screenshot saved to {screenshot_path}")

        except Exception as e:
            print(f"Error: {e}")
            page.screenshot(path="verification/adv_ingest_error.png")
        finally:
            browser.close()

if __name__ == "__main__":
    verify_advanced_ingest_ui()
