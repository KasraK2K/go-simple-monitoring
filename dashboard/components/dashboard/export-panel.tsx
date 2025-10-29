'use client';

import {
  Button,
  HStack,
  IconButton,
  Modal,
  ModalBody,
  ModalCloseButton,
  ModalContent,
  ModalHeader,
  ModalOverlay,
  Text,
  VStack,
  useColorMode,
  useToast,
} from '@chakra-ui/react';
import { FiDownload, FiFile, FiFileText } from 'react-icons/fi';
import { NormalizedMetrics, HeartbeatEntry, AlertMessage } from '@/lib/types';

interface ExportPanelProps {
  isOpen: boolean;
  onClose: () => void;
  metrics?: NormalizedMetrics | null;
  heartbeats?: HeartbeatEntry[];
  alerts?: AlertMessage[];
  series?: any[];
}

export function ExportPanel({ 
  isOpen, 
  onClose, 
  metrics, 
  heartbeats = [], 
  alerts = [], 
  series = [] 
}: ExportPanelProps) {
  const { colorMode } = useColorMode();
  const toast = useToast();

  const downloadFile = (content: string, filename: string, type: string) => {
    const blob = new Blob([content], { type });
    const url = URL.createObjectURL(blob);
    const link = document.createElement('a');
    link.href = url;
    link.download = filename;
    document.body.appendChild(link);
    link.click();
    document.body.removeChild(link);
    URL.revokeObjectURL(url);
  };

  const generateCSV = () => {
    const timestamp = new Date().toISOString().split('T')[0];
    
    let csv = 'Timestamp,Type,Name,Value,Unit\n';
    
    // Add current metrics
    if (metrics) {
      csv += `${new Date().toISOString()},metric,CPU Usage,${metrics.cpu_usage || 0},%\n`;
      csv += `${new Date().toISOString()},metric,Memory Usage,${metrics.memory?.percentage || 0},%\n`;
      csv += `${new Date().toISOString()},metric,Memory Used,${metrics.memory?.used_bytes || 0},bytes\n`;
      csv += `${new Date().toISOString()},metric,Memory Total,${metrics.memory?.total_bytes || 0},bytes\n`;
      csv += `${new Date().toISOString()},metric,Disk Usage,${metrics.disk?.percentage || 0},%\n`;
      csv += `${new Date().toISOString()},metric,Disk Used,${metrics.disk?.used_bytes || 0},bytes\n`;
      csv += `${new Date().toISOString()},metric,Disk Total,${metrics.disk?.total_bytes || 0},bytes\n`;
      
      if (metrics.load_average) {
        csv += `${new Date().toISOString()},metric,Load Average (1m),${metrics.load_average.one_minute || 0},load\n`;
        csv += `${new Date().toISOString()},metric,Load Average (5m),${metrics.load_average.five_minutes || 0},load\n`;
        csv += `${new Date().toISOString()},metric,Load Average (15m),${metrics.load_average.fifteen_minutes || 0},load\n`;
      }

      // Add disk spaces
      if (metrics.disk_spaces) {
        metrics.disk_spaces.forEach((disk, index) => {
          csv += `${new Date().toISOString()},storage,${disk.path || `Disk ${index + 1}`} Usage,${disk.used_pct || 0},%\n`;
          csv += `${new Date().toISOString()},storage,${disk.path || `Disk ${index + 1}`} Used,${disk.used_bytes || 0},bytes\n`;
          csv += `${new Date().toISOString()},storage,${disk.path || `Disk ${index + 1}`} Total,${disk.total_bytes || 0},bytes\n`;
        });
      }
    }

    // Add heartbeats
    heartbeats.forEach(heartbeat => {
      csv += `${new Date().toISOString()},heartbeat,${heartbeat.name || 'Unknown'},${heartbeat.status},status\n`;
      csv += `${new Date().toISOString()},heartbeat,${heartbeat.name || 'Unknown'} Response Time,${heartbeat.last_duration_ms || 0},ms\n`;
    });

    // Add alerts
    alerts.forEach(alert => {
      csv += `${new Date().toISOString()},alert,${alert.type},${alert.message},message\n`;
    });

    downloadFile(csv, `monitoring-data-${timestamp}.csv`, 'text/csv');
    
    toast({
      title: 'Export Successful',
      description: 'Data exported as CSV',
      status: 'success',
      duration: 3000,
      isClosable: true,
    });
    
    onClose();
  };

  const generateJSON = () => {
    const timestamp = new Date().toISOString().split('T')[0];
    
    const exportData = {
      exported_at: new Date().toISOString(),
      timestamp,
      metrics: metrics || null,
      heartbeats: heartbeats || [],
      alerts: alerts || [],
      series_data: series || [],
      metadata: {
        version: '1.0',
        source: 'monitoring-dashboard',
        description: 'System monitoring data export'
      }
    };

    const jsonString = JSON.stringify(exportData, null, 2);
    downloadFile(jsonString, `monitoring-data-${timestamp}.json`, 'application/json');
    
    toast({
      title: 'Export Successful',
      description: 'Data exported as JSON',
      status: 'success',
      duration: 3000,
      isClosable: true,
    });
    
    onClose();
  };

  return (
    <Modal isOpen={isOpen} onClose={onClose} size="md" isCentered>
      <ModalOverlay 
        bg="blackAlpha.300"
        backdropFilter="blur(10px)" 
      />
      <ModalContent
        bg={colorMode === 'dark' ? 'navy.800' : 'white'}
        borderRadius="20px"
        border="1px solid"
        borderColor={colorMode === 'dark' ? 'navy.700' : 'gray.100'}
        boxShadow={colorMode === 'dark' ? 'cardDark' : 'cardLight'}
      >
        <ModalHeader>
          <HStack spacing={3}>
            <IconButton
              aria-label="Export"
              icon={<FiDownload />}
              size="sm"
              variant="ghost"
              color="brand.500"
              isDisabled
            />
            <Text fontWeight="bold" fontSize="lg">
              Export Data
            </Text>
          </HStack>
        </ModalHeader>
        <ModalCloseButton />
        
        <ModalBody pb={6}>
          <VStack spacing={4} align="stretch">
            <Text 
              fontSize="sm" 
              color={colorMode === 'dark' ? 'gray.400' : 'gray.600'}
              mb={2}
            >
              Choose your preferred format to download the current monitoring data.
            </Text>
            
            <Button
              leftIcon={<FiFileText />}
              onClick={generateCSV}
              variant="outline"
              size="lg"
              h="60px"
              justifyContent="flex-start"
              _hover={{
                bg: colorMode === 'dark' ? 'navy.700' : 'gray.50',
                transform: 'translateY(-2px)',
              }}
              transition="all 0.2s"
            >
              <VStack align="flex-start" spacing={0} ml={3}>
                <Text fontWeight="600">Export as CSV</Text>
                <Text fontSize="xs" color="gray.500">
                  Comma-separated values for spreadsheet analysis
                </Text>
              </VStack>
            </Button>
            
            <Button
              leftIcon={<FiFile />}
              onClick={generateJSON}
              variant="outline" 
              size="lg"
              h="60px"
              justifyContent="flex-start"
              _hover={{
                bg: colorMode === 'dark' ? 'navy.700' : 'gray.50',
                transform: 'translateY(-2px)',
              }}
              transition="all 0.2s"
            >
              <VStack align="flex-start" spacing={0} ml={3}>
                <Text fontWeight="600">Export as JSON</Text>
                <Text fontSize="xs" color="gray.500">
                  Structured data format for programmatic access
                </Text>
              </VStack>
            </Button>
          </VStack>
        </ModalBody>
      </ModalContent>
    </Modal>
  );
}

// Trigger button component
interface ExportTriggerProps {
  onOpen: () => void;
}

export function ExportTrigger({ onOpen }: ExportTriggerProps) {
  return (
    <IconButton
      aria-label="Export data"
      icon={<FiDownload />}
      onClick={onOpen}
      variant="ghost"
      size="lg"
      borderRadius="full"
      _hover={{
        bg: 'brand.50',
        color: 'brand.600',
        transform: 'scale(1.05)',
      }}
      _dark={{
        _hover: {
          bg: 'brand.900',
          color: 'brand.300',
        }
      }}
      transition="all 0.2s"
    />
  );
}