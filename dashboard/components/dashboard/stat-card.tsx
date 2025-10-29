'use client';

import { Card, CardBody, Flex, HStack, Icon, Text, Box, useColorMode } from '@chakra-ui/react';
import { ReactNode } from 'react';
import { TrendIndicator, TrendDirection } from './trend-indicator';

interface StatCardProps {
  label: string;
  value: string;
  helper?: string;
  icon?: React.ComponentType;
  accentColor?: string;
  children?: ReactNode;
  variant?: 'default' | 'gradient' | 'glass';
  trend?: {
    direction: TrendDirection;
    value?: number;
    tooltip?: string;
  };
  isLoading?: boolean;
}

export function StatCard({ 
  label, 
  value, 
  helper, 
  icon, 
  accentColor = 'brand.500', 
  children, 
  variant = 'default',
  trend,
  isLoading = false
}: StatCardProps) {
  const { colorMode } = useColorMode();

  const getCardProps = () => {
    switch (variant) {
      case 'gradient':
        return {
          bgGradient: `linear(135deg, ${accentColor}, brand.600)`,
          color: 'white',
          border: 'none',
          boxShadow: 'brand',
        };
      case 'glass':
        return {
          bg: colorMode === 'dark' 
            ? 'rgba(30, 41, 59, 0.8)' 
            : 'rgba(255, 255, 255, 0.1)',
          backdropFilter: 'blur(20px)',
          border: colorMode === 'dark'
            ? '1px solid rgba(51, 65, 85, 0.3)'
            : '1px solid rgba(255, 255, 255, 0.2)',
          boxShadow: 'glass',
        };
      default:
        return {
          bg: colorMode === 'dark' ? 'navy.800' : 'white',
          borderColor: colorMode === 'dark' ? 'navy.700' : 'gray.100',
          boxShadow: colorMode === 'dark' ? 'cardDark' : 'cardLight',
          transition: 'all 0.3s ease',
          _hover: {
            transform: 'translateY(-4px)',
            boxShadow: colorMode === 'dark' ? 'xl' : 'brand',
          },
        };
    }
  };

  return (
    <Card 
      borderRadius="20px" 
      border="1px solid"
      {...getCardProps()}
      overflow="hidden"
      position="relative"
    >
      {variant === 'gradient' && (
        <Box
          position="absolute"
          top="0"
          right="0"
          w="60px"
          h="60px"
          bgGradient="radial(circle, rgba(255,255,255,0.1), transparent 70%)"
          borderRadius="full"
          transform="translate(20px, -20px)"
        />
      )}
      
      <CardBody p={6}>
        <Flex direction="column" gap={4}>
          <HStack spacing={4} align="center" justify="space-between" w="full">
            <HStack spacing={3}>
              {icon && (
                <Flex
                  align="center"
                  justify="center"
                  w={12}
                  h={12}
                  rounded="16px"
                  bg={variant === 'gradient' || variant === 'glass' 
                    ? 'rgba(255, 255, 255, 0.15)' 
                    : `${accentColor}15`
                  }
                  color={variant === 'gradient' || variant === 'glass' 
                    ? 'white' 
                    : accentColor
                  }
                  boxShadow={variant === 'default' ? '0 4px 12px rgba(0,0,0,0.1)' : 'none'}
                >
                  <Icon as={icon} boxSize={6} />
                </Flex>
              )}
              <Text 
                fontSize="sm" 
                textTransform="uppercase" 
                letterSpacing="wider" 
                fontWeight="600"
                color={variant === 'gradient' || variant === 'glass' 
                  ? 'rgba(255, 255, 255, 0.8)' 
                  : colorMode === 'dark' ? 'gray.400' : 'gray.500'
                }
              >
                {label}
              </Text>
            </HStack>
            
            {trend && (
              <TrendIndicator
                direction={trend.direction}
                value={trend.value}
                size="sm"
                tooltip={trend.tooltip}
              />
            )}
          </HStack>
          
          <Text 
            fontSize={{ base: '2xl', md: '3xl' }} 
            fontWeight="bold" 
            color={variant === 'gradient' || variant === 'glass' 
              ? 'white' 
              : colorMode === 'dark' ? 'white' : 'gray.900'
            }
            lineHeight="1.2"
          >
            {value}
          </Text>
          
          {helper && (
            <Text 
              fontSize="sm" 
              color={variant === 'gradient' || variant === 'glass' 
                ? 'rgba(255, 255, 255, 0.7)' 
                : colorMode === 'dark' ? 'gray.400' : 'gray.600'
              }
              fontWeight="500"
            >
              {helper}
            </Text>
          )}
          
          {children}
        </Flex>
      </CardBody>
    </Card>
  );
}
