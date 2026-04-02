from playwright.sync_api import sync_playwright

def verify_file_inspector():
    with sync_playwright() as p:
        browser = p.chromium.launch()
        page = browser.new_page()
        try:
            print("Navigating to http://localhost:3000...")
            page.goto("http://localhost:3000")

            # Wait for initial load
            page.wait_for_selector("#dashboard")

            print("Clicking Files tab...")
            page.click("button[data-tab='files']")

            # Wait for files table
            page.wait_for_selector("#files-table")

            # Wait for Inspect button (🔍)
            print("Waiting for file list...")
            page.wait_for_selector("text=test_video.mp4", timeout=5000)

            # Click Inspect button
            print("Clicking Inspect...")
            # Select the button that contains the 🔍 text or is the second button in the row
            # Since I added it after Play, it might be the second button.
            # But the row structure is <td>...</td>...<td>... <button>Play</button> <button>Inspect</button></td>

            # Use a more specific locator if possible, or just the text
            # Note: The button text is "🔍"
            page.click("button:has-text('🔍')")

            # Wait for Modal
            print("Waiting for Inspector Modal...")
            page.wait_for_selector("#inspector-container:not(.hidden)", timeout=5000)
            page.wait_for_selector("#inspector-title:has-text('Health: test_video.mp4')")

            # Wait for grid population
            page.wait_for_selector("#insp-grid div")

            # Take screenshot
            screenshot_path = "verification/file_inspector.png"
            page.screenshot(path=screenshot_path)
            print(f"Screenshot saved to {screenshot_path}")

        except Exception as e:
            print(f"Error: {e}")
            page.screenshot(path="verification/inspector_error.png")
        finally:
            browser.close()

if __name__ == "__main__":
    verify_file_inspector()
