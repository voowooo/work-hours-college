export interface SwipeHandlers {
  onSwipeLeft?: () => void
  onSwipeRight?: () => void
  threshold?: number
}

export function useSwipe(handlers: SwipeHandlers) {
  const threshold = handlers.threshold || 50
  let touchStartX = 0
  let touchStartY = 0

  function handleTouchStart(e: TouchEvent) {
    touchStartX = e.touches[0]?.clientX || 0
    touchStartY = e.touches[0]?.clientY || 0
  }

  function handleTouchEnd(e: TouchEvent) {
    const touchEndX = e.changedTouches[0]?.clientX || 0
    const touchEndY = e.changedTouches[0]?.clientY || 0
    const diffX = touchStartX - touchEndX
    const diffY = touchStartY - touchEndY

    if (Math.abs(diffX) > Math.abs(diffY) && Math.abs(diffX) > threshold) {
      if (diffX > 0 && handlers.onSwipeLeft) {
        handlers.onSwipeLeft()
      } else if (diffX < 0 && handlers.onSwipeRight) {
        handlers.onSwipeRight()
      }
    }
  }

  return {
    handleTouchStart,
    handleTouchEnd
  }
}
