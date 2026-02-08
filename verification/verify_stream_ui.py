from playwright.sync_api import sync_playwright
import time

def run():
    with sync_playwright() as p:
        browser = p.chromium.launch()
        page = browser.new_page()

        # Navigate
        page.goto("http://localhost:3001")

        # Click Files tab
        page.click("button[data-tab='files']")

        # Wait for table to populate
        page.wait_for_selector("text=test_video.mp4")

        # Check for Play button
        # It's a button with "Play" text inside the table row
        # Selector: tr containing "test_video.mp4" -> button

        # We can just take a screenshot of the files tab
        time.sleep(1) # wait for animation/render

        page.screenshot(path="verification/stream_ui.png")
        print("Screenshot taken")

        browser.close()

if __name__ == "__main__":
    run()
