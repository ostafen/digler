import { useState, useEffect } from "react";
import { motion } from "framer-motion";
import { Folder, HardDrive, Clock, ChevronRight } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { DigleLogo } from "@/components/DigleLogo";

interface RecentScan {
  id: string;
  path: string;
  timestamp: string;
  type: "image" | "device";
  filesFound: number;
}

interface HomeProps {
  onNavigateToScan: () => void;
  onOpenRecent: (scanId: string) => void;
}

export const Home = ({ onNavigateToScan, onOpenRecent }: HomeProps) => {
  const [recentScans, setRecentScans] = useState<RecentScan[]>([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    const fetchRecentScans = async () => {
      try {
        const response = await fetch("/api/recent-scans"); // adjust API path
        if (!response.ok) throw new Error("Failed to fetch scans");
        const data: RecentScan[] = await response.json();
        setRecentScans(data);
      } catch (error) {
        console.error("Error fetching recent scans:", error);
      } finally {
        setLoading(false);
      }
    };

    fetchRecentScans();
  }, []);

  return (
    <div className="min-h-screen bg-gradient-subtle">
      <div className="container mx-auto px-6 py-12 max-w-4xl">
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
          <Card
            className="card-elevated hover:scale-105 transition-smooth cursor-pointer"
            onClick={onNavigateToScan}
          >
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

          <Card
            className="card-elevated hover:scale-105 transition-smooth cursor-pointer"
            onClick={onNavigateToScan}
          >
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
        >
          <Card>
            <CardHeader>
              <CardTitle className="flex items-center gap-2">
                <Clock className="h-5 w-5 text-primary" />
                Recent Scans
              </CardTitle>
            </CardHeader>
            <CardContent>
              {loading ? (
                <p className="text-muted-foreground">Loading recent scans...</p>
              ) : recentScans.length === 0 ? (
                <p className="text-muted-foreground">No recent scans found</p>
              ) : (
                <div className="space-y-3">
                  {recentScans.map((scan) => (
                    <div
                      key={scan.id}
                      className="flex items-center justify-between p-3 rounded-lg bg-muted/30 hover:bg-muted/50 transition-quick cursor-pointer"
                      onClick={() => onOpenRecent(scan.id)}
                    >
                      <div className="flex items-center gap-3">
                        {scan.type === "image" ? (
                          <Folder className="h-4 w-4 text-info" />
                        ) : (
                          <HardDrive className="h-4 w-4 text-info" />
                        )}
                        <div>
                          <p className="font-medium text-sm text-foreground truncate max-w-md">
                            {scan.path}
                          </p>
                          <p className="text-xs text-muted-foreground">
                            {scan.timestamp} • {scan.filesFound.toLocaleString()} files found
                          </p>
                        </div>
                      </div>
                      <ChevronRight className="h-4 w-4 text-muted-foreground" />
                    </div>
                  ))}
                </div>
              )}
            </CardContent>
          </Card>
        </motion.div>
      </div>
    </div>
  );
};
