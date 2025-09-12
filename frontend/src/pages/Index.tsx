import { motion } from "framer-motion";

const Index = () => {
  return (
    <motion.div 
      className="flex min-h-screen items-center justify-center bg-gradient-subtle"
      initial={{ opacity: 0 }}
      animate={{ opacity: 1 }}
    >
      <div className="text-center">
        <motion.div
          initial={{ scale: 0.8, opacity: 0 }}
          animate={{ scale: 1, opacity: 1 }}
          transition={{ delay: 0.2 }}
        >
          <h1 className="mb-4 text-4xl font-bold hero-gradient bg-clip-text text-transparent">
            Digler is Loading...
          </h1>
          <p className="text-xl text-muted-foreground">
            Forensic data recovery interface is starting up
          </p>
        </motion.div>
      </div>
    </motion.div>
  );
};

export default Index;
