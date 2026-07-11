# SAR Tile Download Instructions

Place a pre-downloaded **Sentinel-1 GRD** tile here covering the scripted
vessel's last-dark position (Strait of Malacca, ~lat 3.5 lon 102.0).

## Recommended tile

| Field | Value |
|-------|-------|
| Product type | GRD (Ground Range Detected) |
| Mode | IW (Interferometric Wide) |
| Polarisation | VV+VH |
| Area | Strait of Malacca (lat 1–6.5, lon 99–104.5) |
| Example filename | `S1A_IW_GRDH_1SDV_20260710T123456_...SAFE.zip` |

## Download steps

1. Register (free) at [Copernicus Data Space](https://dataspace.copernicus.eu/)
2. Search:  
   `Platform: Sentinel-1 | Product type: GRD | Area: 3.5N 102.0E | Date: recent`
3. Download the `.SAFE.zip`, unzip, copy the `.tiff` file here and rename to:
   ```
   S1A_IW_GRDH_1SDV_20260710T123456.tiff
   ```
4. Update `SARTile.FilePath` in `cmd/varuna/main.go` if the filename differs.

## Note on the demo framing

The SAR job runs on a slow background ticker (default 30s) and only fires
when the tile's bounding box covers the alert's last-known position.

Say once in the demo:  
*"In production, most alerts never get a satellite match — SAR revisit is
measured in days. We have one pre-downloaded tile that covers today's scripted
vessel. It ran offline and upgraded this alert to corroborated."*
