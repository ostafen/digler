import { useState, useEffect } from "react";
import { motion } from "framer-motion";
import { ArrowLeft, Folder, CheckCircle, AlertTriangle, Download } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { ProgressBar } from "@/components/ProgressBar";
import { DigleLogo } from "@/components/DigleLogo";
import { StartRecovery, RecoveryProgress } from '../wailsjs/go/api/ScanAPI';
import { OpenFolderDialog } from '../wailsjs/go/app/App';
import { GetConfig } from '../wailsjs/go/api/ConfigAPI';

interface FileItem {
  id: string;
  name: string;
  path: string;
  size: string;
  type: string;
  status: string;
}

interface RecoveryProps {
  onBack: () => void;
  onComplete: () => void;
  scanID: string;
  selectedFiles: FileItem[];
}

export const Recovery = ({ onBack, onComplete, scanID, selectedFiles }: RecoveryProps) => {
  const [recoveryPath, setRecoveryPath] = useState("");
  const [isRecovering, setIsRecovering] = useState(false);
  const [progress, setProgress] = useState(0);
  const [currentFile, setCurrentFile] = useState("");
  const [recoveredCount, setRecoveredCount] = useState(0);
  const [errorCount, setErrorCount] = useState(0);
  const [isComplete, setIsComplete] = useState(false);
  const [totalFiles, setTotalFiles] = useState(0);

  const size = selectedFiles.reduce((acc, item) => acc + parseInt(item.size), 0);

  const [totalSize] = useState(size);

  const browseRecoveryPath = async () => {
    try {
      const folderPath = await OpenFolderDialog();
      if (folderPath) setRecoveryPath(folderPath)
    } catch (err) {
      console.error("Folder dialog cancelled or failed", err);
    }
  };

  useEffect(() => {
    GetConfig("OUTDIR").
      then(outDir => setRecoveryPath(outDir))
  }, []);

  useEffect(() => {
    if (isRecovering && !isComplete) {
      const interval = setInterval(async () => {
        const status = await RecoveryProgress(scanID)

        const totalProcessed = status.recovered + status.errors;

        setRecoveredCount(status.recovered);
        setErrorCount(status.errors);
        setTotalFiles(totalProcessed);

        setProgress(prev => {
          const newProgress = status.progress;

          // Update current file
          if (totalProcessed - 1 < selectedFiles.length) {
            setCurrentFile(selectedFiles[totalProcessed - 1].name);
          }

          if (newProgress >= 1) {
            clearInterval(interval);
            setIsComplete(true);
          }
          return newProgress * 100;
        });
      }, 150);

      return () => clearInterval(interval);
    }
  }, [isRecovering, isComplete, totalFiles, selectedFiles, errorCount]);

  const handleStartRecovery = async () => {
    if (!recoveryPath) return;

    const names = selectedFiles.map(file => file.name);

    try {
      await StartRecovery(scanID, names, recoveryPath);
    } catch (err) {
      console.log(`unable to start recovery ${err} `);
      return;
    }
    setIsRecovering(true);
  };

  const handleDownloadLog = () => {
    const logContent = `
Digler Recovery Report
Generated: ${new Date().toLocaleString()}

Recovery Summary:
- Total files processed: ${totalFiles}
- Successfully recovered: ${recoveredCount}
- Errors encountered: ${errorCount}
- Recovery path: ${recoveryPath}

File Details:
${selectedFiles.map(file => `- ${file.name} (${file.size}) - ${file.status}`).join('\n')}
    `.trim();

    const blob = new Blob([logContent], { type: 'text/plain' });
    const url = URL.createObjectURL(blob);
    const a = document.createElement('a');
    a.href = url;
    a.download = 'digler_recovery_report.txt';
    document.body.appendChild(a);
    a.click();
    document.body.removeChild(a);
    URL.revokeObjectURL(url);
  };

  return (
    <div className="min-h-screen bg-gradient-subtle">
      <div className="container mx-auto px-6 py-8 max-w-4xl">
        {/* Header */}
        <motion.div
          className="flex items-center gap-4 mb-8"
          initial={{ opacity: 0, x: -20 }}
          animate={{ opacity: 1, x: 0 }}
        >
          <Button variant="ghost" onClick={onBack} className="p-2" disabled={isRecovering && !isComplete}>
            <ArrowLeft className="h-4 w-4" />
          </Button>
          <DigleLogo size="sm" showTagline={false} />
          <div className="flex-1">
            <h2 className="text-xl font-semibold">File Recovery</h2>
            <p className="text-sm text-muted-foreground">
              {totalFiles} files selected ({totalSize.toFixed(1)} MB total)
            </p>
          </div>
        </motion.div>

        {!isRecovering ? (
          <motion.div
            className="space-y-6"
            initial={{ opacity: 0, y: 20 }}
            animate={{ opacity: 1, y: 0 }}
          >
            {/* Destination Selection */}
            <Card>
              <CardHeader>
                <CardTitle className="flex items-center gap-2">
                  <Folder className="h-5 w-5" />
                  Recovery Destination
                </CardTitle>
                <CardDescription>
                  Choose where to save your recovered files
                </CardDescription>
              </CardHeader>
              <CardContent className="space-y-4">
                <div className="space-y-2">
                  <Label>Output Directory</Label>
                  <div className="flex gap-2">
                    <Input
                      readOnly
                      value={recoveryPath}
                      onChange={(e) => setRecoveryPath(e.target.value)}
                    />
                    <Button variant="outline" onClick={browseRecoveryPath}>Browse</Button>
                  </div>
                </div>
              </CardContent>
            </Card>

            {/* File Summary */}
            <Card>
              <CardHeader>
                <CardTitle>Files to Recover</CardTitle>
              </CardHeader>
              <CardContent>
                <div className="space-y-2 max-h-48 overflow-y-auto">
                  {selectedFiles.map((file) => (
                    <div key={file.id} className="flex items-center justify-between p-2 bg-muted/30 rounded">
                      <span className="text-sm font-medium truncate max-w-md">{file.name}</span>
                      <div className="flex items-center gap-2">
                        <span className="text-xs text-muted-foreground">{file.size}</span>
                        <div className={`h-2 w-2 rounded-full ${file.status === 'recoverable' ? 'bg-success' :
                          file.status === 'partial' ? 'bg-warning' : 'bg-danger'
                          }`} />
                      </div>
                    </div>
                  ))}
                </div>
              </CardContent>
            </Card>

            {/* Start Recovery */}
            <Button
              size="lg"
              className="w-full hero-gradient text-primary-foreground font-semibold"
              onClick={handleStartRecovery}
              disabled={!recoveryPath}
            >
              <Download className="mr-2 h-5 w-5" />
              Start Recovery Process
            </Button>
          </motion.div>
        ) : (
          <motion.div
            className="space-y-6"
            initial={{ opacity: 0, y: 20 }}
            animate={{ opacity: 1, y: 0 }}
          >
            {/* Progress Section */}
            <Card>
              <CardHeader>
                <CardTitle>Recovery in Progress</CardTitle>
                <CardDescription>Saving files to {recoveryPath}</CardDescription>
              </CardHeader>
              <CardContent className="space-y-4">
                <ProgressBar
                  progress={progress}
                  label="Overall Progress"
                  variant={isComplete ? "recovery" : "default"}
                />

                {!isComplete && (
                  <div className="text-center">
                    <p className="text-sm text-muted-foreground">
                      Currently recovering: <span className="font-medium text-foreground">{currentFile}</span>
                    </p>
                  </div>
                )}

                <div className="grid grid-cols-3 gap-4 text-center">
                  <div>
                    <p className="text-2xl font-bold text-primary">{recoveredCount}</p>
                    <p className="text-sm text-muted-foreground">Recovered</p>
                  </div>
                  <div>
                    <p className="text-2xl font-bold text-primary">{totalFiles - recoveredCount - errorCount}</p>
                    <p className="text-sm text-muted-foreground">Remaining</p>
                  </div>
                  <div>
                    <p className="text-2xl font-bold text-warning">{errorCount}</p>
                    <p className="text-sm text-muted-foreground">Errors</p>
                  </div>
                </div>
              </CardContent>
            </Card>

            {/* Completion Summary */}
            {isComplete && (
              <motion.div
                initial={{ opacity: 0, scale: 0.95 }}
                animate={{ opacity: 1, scale: 1 }}
                transition={{ delay: 0.3 }}
              >
                <Card className="border-success/20 bg-success/5">
                  <CardHeader>
                    <CardTitle className="flex items-center gap-2 text-success">
                      <CheckCircle className="h-5 w-5" />
                      Recovery Complete!
                    </CardTitle>
                  </CardHeader>
                  <CardContent className="space-y-4">
                    <div className="space-y-2">
                      <div className="flex justify-between">
                        <span>✅ Files successfully recovered:</span>
                        <span className="font-semibold text-success">{recoveredCount}</span>
                      </div>
                      {errorCount > 0 && (
                        <div className="flex justify-between">
                          <span>⚠️ Files with errors:</span>
                          <span className="font-semibold text-warning">{errorCount}</span>
                        </div>
                      )}
                      <div className="flex justify-between">
                        <span>📁 Recovery location:</span>
                        <span className="font-mono text-sm text-muted-foreground">{recoveryPath}</span>
                      </div>
                    </div>

                    <div className="flex gap-2">
                      <Button
                        variant="outline"
                        onClick={handleDownloadLog}
                        className="flex-1"
                      >
                        <Download className="mr-2 h-4 w-4" />
                        Download Report
                      </Button>
                      <Button
                        onClick={onComplete}
                        className="flex-1 hero-gradient text-primary-foreground"
                      >
                        Return Home
                      </Button>
                    </div>
                  </CardContent>
                </Card>
              </motion.div>
            )}
          </motion.div>
        )}
      </div>
    </div>
  );
};