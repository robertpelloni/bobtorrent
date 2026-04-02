from playwright.sync_api import sync_playwright

def verify_repair_ui():
    with sync_playwright() as p:
        browser = p.chromium.launch()
        page = browser.new_page()
        try:
            # We must mock the /api/files/{id}/health endpoint to simulate a 'Degraded' erasure-coded file
            # to make the 'Repair' button appear.

            def handle_route(route):
                if route.request.url.endswith("/health"):
                    route.fulfill(
                        status=200,
                        content_type="application/json",
                        body='{"fileId": "mock", "status": "Degraded", "totalChunks": 1, "healthyChunks": 0, "erasure": {"dataShards": 4, "parityShards": 2}, "chunks": [{"index": 0, "status": "Degraded", "shards": [{"index": 0, "present": true}, {"index": 1, "present": false}]}]}'
                    )
                else:
                    route.continue_()

            page.route("**/api/files/*/health", handle_route)

            print("Navigating to http://localhost:3000...")
            page.goto("http://localhost:3000")

            # Wait for initial load
            page.wait_for_selector("#dashboard")

            print("Clicking Files tab...")
            page.click("button[data-tab='files']")

            # Wait for files table
            page.wait_for_selector("#files-table")

            # Click Inspect button (first one we find, assuming file exists from previous tests)
            print("Clicking Inspect...")
            page.click("button:has-text('🔍')")

            # Wait for Modal
            print("Waiting for Inspector Modal...")
            page.wait_for_selector("#inspector-container:not(.hidden)")

            # Verify Repair button is visible
            print("Verifying Repair button...")
            page.wait_for_selector("#btn-repair:not(.hidden)")

            # Take screenshot
            screenshot_path = "verification/file_repair_button.png"
            page.screenshot(path=screenshot_path)
            print(f"Screenshot saved to {screenshot_path}")

        except Exception as e:
            print(f"Error: {e}")
            page.screenshot(path="verification/repair_error.png")
        finally:
            browser.close()

if __name__ == "__main__":
    verify_repair_ui()
