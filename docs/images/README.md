# Screenshots

## html-viewer-overview.png

Screenshot requirements:
- Capture the HTML viewer with all panels **collapsed** (not expanded)
- **Redact the AWS Account ID** from the page title (e.g., "AWS Resources (XXXXXXXXXXXX)")
- Include the following visible elements:
  - Control buttons (Expand all, Collapse all, Word wrap, Wide view, Lock to Name, Download all.csv)
  - Category filter input box
  - Multiple collapsed category panels (e.g., acm, cloudformation, dynamodb, ec2, s3, etc.)
  - Each panel should show the "Expand" button
- Recommended size: ~1200px width for good visibility
- Format: PNG with reasonable compression

### How to capture:

1. Start the application and generate HTML output:
   ```bash
   go run cmd/arc/main.go --html
   ```

2. Open the generated HTML in a browser:
   ```bash
   cd output/{account-id}/
   python3 -m http.server 8080
   # Open http://localhost:8080/index.html
   ```

3. Ensure all panels are collapsed

4. Take a screenshot

5. Edit the image to redact the account ID in the title

6. Save as `html-viewer-overview.png` in this directory
