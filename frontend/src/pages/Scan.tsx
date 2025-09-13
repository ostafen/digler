import { useEffect, useRef, useState } from "react";
import { motion } from "framer-motion";
import { ArrowLeft, Folder, HardDrive, Play, Settings, FileText, Pause, Square, RotateCcw } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { Switch } from "@/components/ui/switch";
import { Separator } from "@/components/ui/separator";
import { AlertDialog, AlertDialogAction, AlertDialogCancel, AlertDialogContent, AlertDialogDescription, AlertDialogFooter, AlertDialogHeader, AlertDialogTitle, AlertDialogTrigger } from "@/components/ui/alert-dialog";
import { ProgressBar } from "@/components/ProgressBar";
import { DigleLogo } from "@/components/DigleLogo";
import { OpenFileDialog, OpenFolderDialog } from '../wailsjs/go/app/App';
import { api } from '../wailsjs/go/models.ts';
import { DefaultOutputDir, ListDevices } from '../wailsjs/go/api/SystemAPI';
import { StartScan, PollStatus, PauseScan, ResumeScan, AbortScan } from '../wailsjs/go/api/ScanAPI';
import { GetOrSet, SetConfig } from '../wailsjs/go/api/ConfigAPI';
import { formatFileSize } from '../lib/utils';

const OUT_DIR_CONFIG_KEY = "OUTDIR"

interface ScanProps {
  mode: "image" | "device";
  onBack: () => void;
  onScanComplete: (results: { filesFound: number, path: string, scanId: string }) => void;
}

export const Scan = ({ onBack, onScanComplete, mode }: ScanProps) => {
  const [scanType, setScanTypeState] = useState<"image" | "device">(mode);
  const [selectedPath, setSelectedPath] = useState("");
  const [outputPath, setOutputPath] = useState("");
  const [dumpEnabled, setDumpEnabled] = useState(false);
  const [selectedPlugin, setSelectedPlugin] = useState("");
  const [isScanning, setIsScanning] = useState(false);
  const [isPaused, setIsPaused] = useState(false);
  const [isAborted, setIsAborted] = useState(false);
  const [progress, setProgress] = useState(0);
  const [scanLogs, setScanLogs] = useState<string[]>([]);
  const [scanStats, setScanStats] = useState({ filesFound: 0, timeElapsed: "00:00" });
  const [devices, setDevices] = useState<api.DeviceInfo[]>([]);
  const [errorMessage, setErrorMessage] = useState<string | null>(null);

  const logEndRef = useRef<HTMLDivElement | null>(null);

  useEffect(() => {
    if (logEndRef.current) {
      logEndRef.current.scrollIntoView({ behavior: "smooth" });
    }
  }, [scanLogs]);

  const setErrorMessageWithTimeout = (msg: string, delay: number) => {
    setErrorMessage(msg)
    setTimeout(() => {
      setErrorMessage(null)
    }, delay)
  }

  const setScanType = (type: "image" | "device") => {
    setSelectedPath("");
    setScanTypeState(type);
  }

  // Browse for disk image using Wails native dialog
  const browseFile = async () => {
    try {
      const filters = [
        { name: "Disk Images", pattern: "*.dd;*.img;*.iso" },
        { name: "All Files", pattern: "*" },
      ]

      const filePath = await OpenFileDialog("Select Image", filters);
      if (filePath) setSelectedPath(filePath);
    } catch (err) {
      console.error("File dialog cancelled or failed", err);
    }
  };

  // Browse for output folder using Wails native dialog
  const browseOutputPath = async () => {
    try {
      const folderPath = await OpenFolderDialog();
      if (folderPath) {
        await SetConfig(OUT_DIR_CONFIG_KEY, folderPath);
        setOutputPath(folderPath);
      }
    } catch (err) {
      console.error("Folder dialog cancelled or failed", err);
    }
  };

  useEffect(() => {
    // Fetch list of devices on mount
    const fetchDevices = async () => {
      try {
        const deviceList = await ListDevices();
        setDevices(deviceList);
      } catch (err) {
        console.error("Failed to list devices", err);
      }
    };

    const setDefaultOutputPath = () => {
      DefaultOutputDir().
        then(defaultOutputDir => GetOrSet(OUT_DIR_CONFIG_KEY, defaultOutputDir))
        .then(outDir => setOutputPath(outDir))
        .catch(err => console.error("Failed to get working directory", err))
    }

    fetchDevices();
    setDefaultOutputPath();
  }, []);

  const plugins = [
    "All Formats",
    "Images Only (JPEG, PNG, GIF)",
    "Documents (PDF, DOC, TXT)",
    "Archives (ZIP, RAR, 7Z)",
    "Custom Signatures"
  ];

  const startTimeRef = useRef<number | null>(null);

  const elapsedTime = () => {
    const elapsedMs = Date.now() - startTimeRef.current;

    const totalSeconds = Math.floor(elapsedMs / 1000);
    const hours = Math.floor(totalSeconds / 3600);
    const seconds = totalSeconds % 60;

    const formattedTime = `${String(hours).padStart(2, '0')}:${String(seconds).padStart(2, '0')}`;
    return formattedTime;
  }

  const intervalRef = useRef<number | null>(null);

  const refreshStatus = async () => {
    let scanStatus: api.ScanStatusResponse = null;
    try {
      scanStatus = await PollStatus(scanIdRef.current);
    } catch (error) {
      console.error("Error polling scan status:", error);
      setScanLogs(prev => [...prev, "Error retrieving scan status. Retrying..."]);
      return;
    }

    if (!scanStatus) return;

    setProgress(prev => {
      setScanStats({
        filesFound: scanStatus!.files,
        timeElapsed: elapsedTime()
      });

      const newProgress = scanStatus.progress;

      if (newProgress >= 1) {
        if (intervalRef.current != null)
          clearInterval(intervalRef.current);

        setScanLogs(prev => [...prev, "Scan completed successfully!", "Analyzing results..."]);
        setTimeout(() => {
          onScanComplete(
            {
              filesFound: scanStatus.files,
              path: selectedPath,
              scanId: scanIdRef.current
            }
          );
        }, 2000);
      } else {
        const logs = [
          `Scanning sector ${Math.floor(newProgress * 1000)}...`,
          `Found deleted file: document_${Math.floor(Math.random() * 1000)}.pdf`,
          `Recovered image: IMG_${Math.floor(Math.random() * 9999)}.jpg`,
          `Processing filesystem metadata...`
        ];
        setScanLogs(prev => [...prev, logs[Math.floor(Math.random() * logs.length)]]);
      }
      return newProgress * 100;
    });
  }

  const scanIdRef = useRef<string>("");

  const handleStartScan = async () => {
    if (!selectedPath || !outputPath) return;

    try {
      scanIdRef.current = await StartScan(selectedPath, outputPath);
    } catch (error) {
      let errMsg = error as string
      if (errMsg.includes('permission denied')) {
        errMsg += '\n (Try restarting with elevated privileges)'
      }

      setErrorMessageWithTimeout(errMsg, 3000);
      setScanLogs(prev => [...prev, "Error starting scan. Please try again."]);
      return;
    }

    startTimeRef.current = Date.now()
    intervalRef.current = (setInterval(refreshStatus, 200) as unknown) as number;

    setIsScanning(true);
    setIsPaused(false);
    setIsAborted(false);
    setScanLogs(["Starting forensic scan...", `Target: ${selectedPath}`, `Output: ${outputPath}`]);
  };

  const handlePauseScan = async () => {
    await PauseScan(scanIdRef.current)

    if (intervalRef.current != null)
      clearInterval(intervalRef.current)

    setIsPaused(true);
    setScanLogs(prev => [...prev, "Scan paused by user"]);
  };

  const handleResumeScan = async () => {
    await ResumeScan(scanIdRef.current)

    intervalRef.current = (setInterval(refreshStatus, 200) as unknown) as number;

    setIsPaused(false);
    setScanLogs(prev => [...prev, "Scan resumed"]);
  };

  const handleAbortScan = async () => {
    await AbortScan(scanIdRef.current)

    setIsAborted(true);
    setIsScanning(false);
    setIsPaused(false);
    setScanLogs(prev => [...prev, "Scan aborted by user"]);
  };

  const handleReturnToConfig = () => {
    setIsAborted(false);
    setProgress(0);
    setScanLogs([]);
    setScanStats({ filesFound: 0, timeElapsed: "00:00" });
  };

  const handleViewPartialResults = () => {
    onScanComplete({
      filesFound: scanStats.filesFound,
      path: selectedPath,
      scanId: scanIdRef.current
    });
  }

  return (
    <div className="min-h-screen bg-gradient-subtle">
      <div className="container mx-auto px-6 py-8 max-w-4xl">
        {/* Header */}
        <motion.div
          className="flex items-center gap-4 mb-8"
          initial={{ opacity: 0, x: -20 }}
          animate={{ opacity: 1, x: 0 }}
        >
          {!isScanning && (
            <Button variant="ghost" onClick={onBack} className="p-2">
              <ArrowLeft className="h-4 w-4" />
            </Button>
          )}
          <DigleLogo size="sm" showTagline={false} />
          <div className="flex-1" />
          <span className="text-sm text-muted-foreground">Forensic Data Recovery</span>
        </motion.div>

        {!isScanning && !isAborted ? (
          <motion.div
            className="space-y-6"
            initial={{ opacity: 0, y: 20 }}
            animate={{ opacity: 1, y: 0 }}
          >
            {/* Source Selection */}
            <Card>
              <CardHeader>
                <CardTitle className="flex items-center gap-2">
                  {scanType === "image" ? <Folder className="h-5 w-5" /> : <HardDrive className="h-5 w-5" />}
                  Select Source
                </CardTitle>
                <CardDescription>
                  Choose what you want to scan for deleted or lost files
                </CardDescription>
              </CardHeader>
              <CardContent className="space-y-4">
                <div className="grid grid-cols-2 gap-4">
                  <Button
                    variant={scanType === "image" ? "default" : "outline"}
                    className="h-20 flex-col gap-2"
                    onClick={() => setScanType("image")}
                  >
                    <Folder className="h-6 w-6" />
                    Disk Image
                  </Button>
                  <Button
                    variant={scanType === "device" ? "default" : "outline"}
                    className="h-20 flex-col gap-2"
                    onClick={() => setScanType("device")}
                  >
                    <HardDrive className="h-6 w-6" />
                    Physical Device
                  </Button>
                </div>

                <div className="space-y-2">
                  <Label>{scanType === "image" ? "Image File Path" : "Device"}</Label>
                  {scanType === "image" ? (
                    <div className="flex gap-2">
                      <Input
                        placeholder="Select an image file"
                        readOnly
                        value={selectedPath}
                        onChange={(e) => setSelectedPath(e.target.value)}
                      />
                      <Button variant="outline" onClick={browseFile}>Browse</Button>
                    </div>
                  ) : (
                    <Select value={selectedPath} onValueChange={setSelectedPath}>
                      <SelectTrigger>
                        <SelectValue placeholder="Select device to scan" />
                      </SelectTrigger>
                      <SelectContent>
                        {devices.map((device) => (
                          <SelectItem key={device.name} value={device.path}>
                            {`${device.name} - ${device.model} (${formatFileSize(device.size)})`}
                          </SelectItem>
                        ))}
                      </SelectContent>
                    </Select>
                  )}
                </div>
              </CardContent>
            </Card>

            {/* Output & Options */}
            <Card>
              <CardHeader>
                <CardTitle className="flex items-center gap-2">
                  <Settings className="h-5 w-5" />
                  Scan Options
                </CardTitle>
              </CardHeader>
              <CardContent className="space-y-4">
                <div className="space-y-2">
                  <Label>Output Directory</Label>
                  <div className="flex gap-2">
                    <Input
                      placeholder="Select an output path"
                      readOnly
                      value={outputPath}
                    />
                    <Button variant="outline" onClick={browseOutputPath}>Browse</Button>
                  </div>
                </div>

                <Separator />

                <div className="flex items-center justify-between">
                  <div className="space-y-1">
                    <Label>Recover files during scan</Label>
                    <p className="text-sm text-muted-foreground">
                      Automatically save recovered files (slower but convenient)
                    </p>
                  </div>
                  <Switch checked={dumpEnabled} onCheckedChange={setDumpEnabled} />
                </div>

                <div className="space-y-2">
                  <Label>File Type Filter</Label>
                  <Select value={selectedPlugin} onValueChange={setSelectedPlugin}>
                    <SelectTrigger>
                      <SelectValue placeholder="All file types" />
                    </SelectTrigger>
                    <SelectContent>
                      {plugins.map((plugin) => (
                        <SelectItem key={plugin} value={plugin}>
                          {plugin}
                        </SelectItem>
                      ))}
                    </SelectContent>
                  </Select>
                </div>
              </CardContent>
            </Card>

            {/* Start Button */}
            <div className="space-y-3">
              <Button
                size="lg"
                className="w-full hero-gradient text-primary-foreground font-semibold"
                onClick={handleStartScan}
                disabled={!selectedPath || !outputPath}
              >
                <Play className="mr-2 h-5 w-5" />
                Start Deep Scan
              </Button>

              {/* Error/Success Message */}
              {errorMessage && (
                <motion.div
                  initial={{ opacity: 0, y: -10 }}
                  animate={{ opacity: 1, y: 0 }}
                  exit={{ opacity: 0, y: -10 }}
                  className="text-center text-sm font-medium text-danger"
                >
                  {errorMessage}
                </motion.div>
              )}
            </div>
          </motion.div>
        ) : isScanning ? (
          <motion.div
            className="space-y-6"
            initial={{ opacity: 0, y: 20 }}
            animate={{ opacity: 1, y: 0 }}
          >
            {/* Progress Section */}
            <Card>
              <CardHeader>
                <CardTitle>Scanning in Progress</CardTitle>
                <CardDescription>Deep scanning {selectedPath}</CardDescription>
              </CardHeader>
              <CardContent className="space-y-4">
                <ProgressBar
                  progress={Math.trunc(progress * 100) / 100}
                  label="Overall Progress"
                  showPercentage={true}
                  variant="default"
                />

                <div className="grid grid-cols-3 gap-4 text-center">
                  <div>
                    <p className="text-2xl font-bold text-primary">{scanStats.filesFound}</p>
                    <p className="text-sm text-muted-foreground">Files Found</p>
                  </div>
                  <div>
                    <p className="text-2xl font-bold text-primary">{scanStats.timeElapsed}</p>
                    <p className="text-sm text-muted-foreground">Time Elapsed</p>
                  </div>
                  <div>
                    <p className="text-2xl font-bold text-primary">{Math.round(progress)}%</p>
                    <p className="text-sm text-muted-foreground">Complete</p>
                  </div>
                </div>

                {/* Scan Control Buttons */}
                <div className="flex gap-2 justify-center mt-6">
                  {!isPaused ? (
                    <Button variant="outline" onClick={handlePauseScan}>
                      <Pause className="mr-2 h-4 w-4" />
                      Pause
                    </Button>
                  ) : (
                    <Button variant="outline" onClick={handleResumeScan}>
                      <RotateCcw className="mr-2 h-4 w-4" />
                      Resume
                    </Button>
                  )}

                  <AlertDialog>
                    <AlertDialogTrigger asChild>
                      <Button variant="destructive">
                        <Square className="mr-2 h-4 w-4" />
                        Abort
                      </Button>
                    </AlertDialogTrigger>
                    <AlertDialogContent>
                      <AlertDialogHeader>
                        <AlertDialogTitle>Abort Scan</AlertDialogTitle>
                        <AlertDialogDescription>
                          Are you sure you want to abort the scan? This will stop the current operation and you can view partial results or restart the scan.
                        </AlertDialogDescription>
                      </AlertDialogHeader>
                      <AlertDialogFooter>
                        <AlertDialogCancel>Cancel</AlertDialogCancel>
                        <AlertDialogAction onClick={handleAbortScan}>Abort Scan</AlertDialogAction>
                      </AlertDialogFooter>
                    </AlertDialogContent>
                  </AlertDialog>
                </div>
              </CardContent>
            </Card>

            {/* Live Log */}
            <Card>
              <CardHeader>
                <CardTitle className="flex items-center gap-2">
                  <FileText className="h-5 w-5" />
                  Live Scan Log
                </CardTitle>
              </CardHeader>
              <CardContent>
                <div className="bg-background border rounded-lg p-4 h-64 overflow-y-auto font-mono text-sm">
                  {scanLogs.map((log, index) => (
                    <div key={index} className="text-foreground">
                      <span className="text-muted-foreground mr-2">
                        [{new Date().toLocaleTimeString()}]
                      </span>
                      {log}
                    </div>
                  ))}

                  {/* Invisible div to scroll into view */}
                  <div ref={logEndRef} />
                </div>
              </CardContent>
            </Card>
          </motion.div>
        ) : isAborted ? (
          <motion.div
            className="space-y-6"
            initial={{ opacity: 0, y: 20 }}
            animate={{ opacity: 1, y: 0 }}
          >
            {/* Post-Abort Options */}
            <Card>
              <CardHeader>
                <CardTitle>Scan Aborted</CardTitle>
                <CardDescription>
                  The scan was stopped. You can view the partial results collected so far or return to configure a new scan.
                </CardDescription>
              </CardHeader>
              <CardContent className="space-y-4">
                <div className="text-center space-y-2">
                  <p className="text-lg font-semibold">Partial Results Available</p>
                  <p className="text-2xl font-bold text-primary">{scanStats.filesFound} files found</p>
                  <p className="text-sm text-muted-foreground">before scan was aborted</p>
                </div>

                <div className="flex gap-4 justify-center">
                  <Button variant="outline" onClick={handleReturnToConfig}>
                    <Settings className="mr-2 h-4 w-4" />
                    Configure New Scan
                  </Button>
                  <Button onClick={handleViewPartialResults}>
                    <FileText className="mr-2 h-4 w-4" />
                    View Partial Results
                  </Button>
                </div>
              </CardContent>
            </Card>
          </motion.div>
        ) : null}
      </div>
    </div>
  );
};
