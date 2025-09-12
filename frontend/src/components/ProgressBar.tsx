import { motion } from "framer-motion";

interface ProgressBarProps {
  progress: number;
  label?: string;
  showPercentage?: boolean;
  variant?: "default" | "recovery" | "warning" | "danger";
  className?: string;
}

export const ProgressBar = ({ 
  progress, 
  label, 
  showPercentage = true, 
  variant = "default",
  className = "" 
}: ProgressBarProps) => {
  const getVariantClasses = () => {
    switch (variant) {
      case "recovery":
        return "bg-success progress-glow";
      case "warning":
        return "bg-warning";
      case "danger":
        return "bg-danger";
      default:
        return "bg-primary progress-glow";
    }
  };

  return (
    <div className={`w-full ${className}`}>
      {label && (
        <div className="flex justify-between items-center mb-2">
          <span className="text-sm font-medium text-foreground">{label}</span>
          {showPercentage && (
            <span className="text-sm text-muted-foreground">{progress}%</span>
          )}
        </div>
      )}
      <div className="w-full bg-muted rounded-full h-2 overflow-hidden">
        <motion.div
          className={`h-full rounded-full transition-smooth ${getVariantClasses()}`}
          initial={{ width: 0 }}
          animate={{ width: `${progress}%` }}
          transition={{ duration: 0.5, ease: "easeOut" }}
        />
      </div>
    </div>
  );
};