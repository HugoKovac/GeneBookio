import { Capacitor } from '@capacitor/core';
import { LOG_LEVEL, Purchases, type PurchasesPackage } from '@revenuecat/purchases-capacitor';

let configurePromise: Promise<void> | null = null;
let configuredUserID: string | null = null;

// Ties every native purchase back to our own user, mirroring how the
// Stripe checkout session sets SubscriptionData.Metadata["user_id"] — see
// backend/internal/subscription/stripe_client.go. Safe to call repeatedly —
// getCurrentOffering/purchaseCurrentPackage await the in-flight configure
// call rather than assuming it has already finished, since the native SDK
// bridge call is asynchronous even though this function itself isn't. If
// the SDK is already configured for a *different* user (e.g. logout then
// login as another account in the same app session), switches identity via
// logIn() instead of no-op'ing — otherwise RevenueCat stays pinned to the
// first user's appUserID and later reconcile calls 404 ("resource_missing")
// because RevenueCat has no customer record for the new user's id.
export function configureRevenueCat(userID: string) {
  if (!Capacitor.isNativePlatform() || configuredUserID === userID) return;

  if (configurePromise) {
    configurePromise = configurePromise
      .then(() => Purchases.logIn({ appUserID: userID }))
      .then(() => {
        configuredUserID = userID;
        console.log('[RevenueCat] switched to appUserID', userID);
      })
      .catch((cause) => {
        configuredUserID = null; // allow a retry on the next call rather than getting stuck
        console.error('[RevenueCat] logIn() failed', cause);
        throw cause;
      });
    return;
  }

  const apiKey = Capacitor.getPlatform() === 'ios'
    ? import.meta.env.VITE_REVENUECAT_IOS_API_KEY
    : import.meta.env.VITE_REVENUECAT_ANDROID_API_KEY;

  if (!apiKey) {
    // Logged loudly (not silently skipped) since a missing key here means
    // every purchase call downstream will fail with the SDK's generic
    // "Purchases must be configured" error, which doesn't point back to
    // this as the cause on its own.
    console.error('[RevenueCat] no API key found for platform', Capacitor.getPlatform(), '— check .env.native was baked into this build');
    return;
  }

  Purchases.setLogLevel({ level: LOG_LEVEL.DEBUG });

  configurePromise = Purchases.configure({ apiKey, appUserID: userID })
    .then(() => {
      configuredUserID = userID;
      console.log('[RevenueCat] configured for platform', Capacitor.getPlatform(), 'appUserID', userID);
    })
    .catch((cause) => {
      configurePromise = null; // allow a retry on the next call rather than getting stuck
      console.error('[RevenueCat] configure() failed', cause);
      throw cause;
    });
}

async function ensureConfigured() {
  if (configurePromise) await configurePromise;
}

export async function getCurrentOffering() {
  await ensureConfigured();
  const offerings = await Purchases.getOfferings();
  return offerings.current;
}

export async function purchaseCurrentPackage(aPackage: PurchasesPackage) {
  await ensureConfigured();
  return Purchases.purchasePackage({ aPackage });
}

export async function restorePurchases() {
  await ensureConfigured();
  return Purchases.restorePurchases();
}
