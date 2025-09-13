import { useState, useEffect } from "react";
import { motion } from "framer-motion";
import { Folder, HardDrive, Clock, ChevronRight, AlertCircle, Trash2 } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { DigleLogo } from "@/components/DigleLogo";
import { Badge } from "@/components/ui/badge";
import { ScrollArea } from "@/components/ui/scroll-area";

import { api } from '../wailsjs/go/models';
import { LoadScanHistory, SetCurrentScan, ClearScanHistory } from '../wailsjs/go/api/ScanAPI';

const MAX_HISTORY_ITEMS = 20

interface HomeProps {
  onNavigateToScan: (mode: 'image' | 'device') => void;
  onOpenRecent: (scanId: string) => void;
}

const formatTime = (timestampSeconds: number): string => {
  const date = new Date(timestampSeconds * 1000);

  // Convert ISO string to "YYYY-MM-DD HH:MM:SS"
  const formatted = date.toISOString().replace('T', ' ').split('.')[0];
  return formatted
}

export const Home = ({ onNavigateToScan, onOpenRecent }: HomeProps) => {
  const [recentScans, setRecentScans] = useState<api.ScanHistoryRecord[]>([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    const fetchRecentScans = async () => {
      try {
        const scanInfo = await LoadScanHistory(MAX_HISTORY_ITEMS);
        setRecentScans(scanInfo);
      } catch (error) {
        console.error("Error fetching recent scans:", error);
      } finally {
        setLoading(false);
      }
    };

    const setCurrentScan = async (scanID: string) => {
      try {
        await SetCurrentScan(scanID)
      } catch (error) {
        console.log("SetCurrentScan: ", error)
      }
    }

    setCurrentScan("")
    fetchRecentScans();
  }, []);

  return (
    <div className="h-screen bg-gradient-subtle overflow-hidden">
      <div className="container mx-auto px-6 py-12 max-w-4xl h-full flex flex-col">
        {/* Hero Section */}
        <motion.div
          className="text-center mb-12"
          initial={{ opacity: 0, y: 20 }}
          animate={{ opacity: 1, y: 0 }}
          transition={{ duration: 0.6 }}
        >
          <DigleLogo size="lg" showTagline={true} />
          <p className="mt-6 text-lg text-muted-foreground max-w-2xl mx-auto">
            Professional forensic data recovery made simple. Recover lost files from disk images and devices with advanced deep-scan technology.
          </p>
        </motion.div>

        {/* Main Actions */}
        <motion.div
          className="grid md:grid-cols-2 gap-6 mb-12"
          initial={{ opacity: 0, y: 20 }}
          animate={{ opacity: 1, y: 0 }}
          transition={{ duration: 0.6, delay: 0.2 }}
        >
          <Card className="card-elevated hover:scale-105 transition-smooth cursor-pointer"
            onClick={() => onNavigateToScan('image')}>
            <CardHeader className="text-center pb-4">
              <div className="mx-auto mb-4 h-16 w-16 rounded-full bg-primary/10 flex items-center justify-center">
                <Folder className="h-8 w-8 text-primary" />
              </div>
              <CardTitle>Open Disk Image</CardTitle>
              <CardDescription>
                Scan .dd, .img, .raw, and other disk image files
              </CardDescription>
            </CardHeader>
            <CardContent className="pt-0">
              <Button className="w-full hero-gradient border-0 text-primary-foreground font-semibold">
                Browse Image Files
                <ChevronRight className="ml-2 h-4 w-4" />
              </Button>
            </CardContent>
          </Card>

          <Card className="card-elevated hover:scale-105 transition-smooth cursor-pointer"
            onClick={() => onNavigateToScan('device')}>
            <CardHeader className="text-center pb-4">
              <div className="mx-auto mb-4 h-16 w-16 rounded-full bg-primary/10 flex items-center justify-center">
                <HardDrive className="h-8 w-8 text-primary" />
              </div>
              <CardTitle>Scan Device</CardTitle>
              <CardDescription>
                Directly scan physical drives and partitions
              </CardDescription>
            </CardHeader>
            <CardContent className="pt-0">
              <Button className="w-full hero-gradient border-0 text-primary-foreground font-semibold">
                Select Device
                <ChevronRight className="ml-2 h-4 w-4" />
              </Button>
            </CardContent>
          </Card>
        </motion.div>

        {/* Recent Scans */}

        <motion.div
          initial={{ opacity: 0, y: 20 }}
          animate={{ opacity: 1, y: 0 }}
          transition={{ duration: 0.6, delay: 0.4 }}
          className="flex-1 min-h-0"
        >
          <Card className="h-full flex flex-col">
            <CardHeader className="flex-shrink-0">
              <CardTitle className="flex items-center justify-between">
                <div className="flex items-center gap-2">
                  <Clock className="h-5 w-5 text-primary" />
                  Recent Scans
                </div>

                {recentScans.length > 0 &&
                  <Button
                    variant="ghost"
                    size="sm"
                    onClick={() => ClearScanHistory().then(() => setRecentScans([]))}
                    className="h-8 w-8 p-0 opacity-60 hover:opacity-100"
                  >
                    <Trash2 className="h-4 w-4" />
                  </Button>
                }
              </CardTitle>
            </CardHeader>
            <CardContent className="flex-1 min-h-0">
              <ScrollArea className="h-full">
                <div className="space-y-3 pr-4">
                  {loading ? (
                    <p className="text-muted-foreground">Loading recent scans...</p>
                  ) : recentScans.length === 0 ? (
                    <p className="text-muted-foreground">No recent scans found</p>
                  ) : recentScans.map((scan) => (
                    <div
                      key={scan.id}
                      className={`flex items-center justify-between p-3 rounded-lg transition-quick relative ${scan.isMissing
                        ? "bg-muted/20 cursor-not-allowed opacity-60"
                        : "bg-muted/30 hover:bg-muted/50 cursor-pointer"
                        }`}
                      onClick={
                        async () => {
                          if (scan.isMissing) return

                          await SetCurrentScan(scan.id)
                          onOpenRecent(scan.id)
                        }
                      }
                    >
                      <div className="flex items-center gap-3 flex-1">
                        {scan.sourceType === "image" ? (
                          <Folder className={`h-4 w-4 ${scan.isMissing ? "text-muted-foreground" : "text-info"}`} />
                        ) : (
                          <HardDrive className={`h-4 w-4 ${scan.isMissing ? "text-muted-foreground" : "text-info"}`} />
                        )}
                        <div className="flex-1 min-w-0">
                          <p className={`font-medium text-sm truncate max-w-md ${scan.isMissing ? "text-muted-foreground" : "text-foreground"
                            }`}>
                            {scan.sourcePath}
                          </p>
                          <p className="text-xs text-muted-foreground">
                            {formatTime(scan.scanStartedAt)} • {scan.filesFound.toLocaleString()} files found
                          </p>
                        </div>
                        {scan.isMissing && (
                          <Badge variant="destructive" className="flex items-center gap-1 text-xs">
                            <AlertCircle className="h-3 w-3" />
                            Missing file
                          </Badge>
                        )}
                      </div>
                      {!scan.isMissing && (
                        <ChevronRight className="h-4 w-4 text-muted-foreground flex-shrink-0" />
                      )}
                    </div>
                  ))}
                </div>
              </ScrollArea>
            </CardContent>
          </Card>
        </motion.div>

      </div>
    </div >
  );
};