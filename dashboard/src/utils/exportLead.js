export function exportLeadPackage(alert) {
  const payload = {
    disclaimer: "investigative lead — not legal evidence",
    exported_at: new Date().toISOString(),
    alert: alert
  };

  const blob = new Blob([JSON.stringify(payload, null, 2)], { type: "application/json" });
  const url = URL.createObjectURL(blob);
  
  const a = document.createElement('a');
  a.href = url;
  a.download = `lead-package-${alert.vessel_id}-${alert.timestamp}.json`;
  document.body.appendChild(a);
  a.click();
  
  // Cleanup
  setTimeout(() => {
    document.body.removeChild(a);
    URL.revokeObjectURL(url);
  }, 0);
}
