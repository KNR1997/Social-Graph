import type React from 'react';
import { cn } from '@/lib/utils';
import { InteractiveGridPattern } from './interactive-grid';

/**
 * The split-screen shell the sign-in and sign-up pages share.
 *
 * Both pages rendered this markup around a Clerk component before; with the
 * forms now being ours, the chrome is worth factoring out rather than keeping
 * two copies that drift.
 */
export default function AuthLayout({
  title,
  subtitle,
  children
}: {
  title: string;
  subtitle: string;
  children: React.ReactNode;
}) {
  return (
    <div className='relative flex min-h-screen flex-col items-center justify-center overflow-hidden md:grid lg:max-w-none lg:grid-cols-2 lg:px-0'>
      <div className='relative hidden h-full flex-col p-10 lg:flex dark:border-r'>
        <div className='bg-sidebar absolute inset-0' />
        <div className='text-sidebar-foreground relative z-20 flex items-center text-lg font-medium'>
          <svg
            xmlns='http://www.w3.org/2000/svg'
            viewBox='0 0 24 24'
            fill='none'
            stroke='currentColor'
            strokeWidth='2'
            strokeLinecap='round'
            strokeLinejoin='round'
            className='mr-2 h-6 w-6'
            aria-hidden
          >
            <path d='M15 6v12a3 3 0 1 0 3-3H6a3 3 0 1 0 3 3V6a3 3 0 1 0-3 3h12a3 3 0 1 0-3-3' />
          </svg>
          Logo
        </div>
        <InteractiveGridPattern
          className={cn(
            'mask-[radial-gradient(400px_circle_at_center,white,transparent)]',
            'inset-x-0 inset-y-[0%] h-full skew-y-12'
          )}
        />
        <div className='text-sidebar-foreground relative z-20 mt-auto'>
          <blockquote className='space-y-2'>
            <p className='text-lg'>
              &ldquo;This starter template has saved me countless hours of work and helped me
              deliver projects to my clients faster than ever before.&rdquo;
            </p>
            <footer className='text-sidebar-foreground/70 text-sm'>Random Dude</footer>
          </blockquote>
        </div>
      </div>
      <div className='flex h-full items-center justify-center p-4 lg:p-8'>
        <div className='flex w-full max-w-sm flex-col justify-center space-y-6'>
          <div className='space-y-2 text-center'>
            <h1 className='text-2xl font-semibold tracking-tight'>{title}</h1>
            <p className='text-muted-foreground text-sm'>{subtitle}</p>
          </div>
          {children}
        </div>
      </div>
    </div>
  );
}
