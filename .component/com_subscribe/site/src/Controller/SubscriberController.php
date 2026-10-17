<?php
/**
 * @version    CVS: 1.0.0
 * @package    Com_Subscription
 * @author     Rob <pr@floridarangers.com>
 * @copyright  2024 Rob
 * @license    GNU General Public License version 2 or later; see LICENSE.txt
 */

namespace Sub\Component\Subscription\Site\Controller;

\defined('_JEXEC') or die;

use \Joomla\CMS\Application\SiteApplication;
use Sub\Component\Subscription\Site\Helper\SubscriptionHelper;
use \Joomla\CMS\Factory;
use \Joomla\CMS\Language\Multilanguage;
use \Joomla\CMS\Language\Text;
use \Joomla\CMS\MVC\Controller\BaseController;
use \Joomla\CMS\Router\Route;
use \Joomla\CMS\Uri\Uri;
use \Joomla\Utilities\ArrayHelper;
require_once(JPATH_ADMINISTRATOR."/components/com_payage/helpers/Stripe/init.php");


/**
 * Subscriber class.
 *
 * @since  1.6.0
 */
class SubscriberController extends BaseController
{
	/**
	 * Method to check out an item for editing and redirect to the edit form.
	 *
	 * @return  void
	 *
	 * @since   1.0.0
	 *
	 * @throws  Exception
	 */
	public function edit()
	{
		// Get the previous edit id (if any) and the current edit id.
		$previousId = (int) $this->app->getUserState('com_subscription.edit.subscriber.id');
		$editId     = $this->input->getInt('id', 0);

		// Set the user id for the user to edit in the session.
		$this->app->setUserState('com_subscription.edit.subscriber.id', $editId);

		// Get the model.
		$model = $this->getModel('Subscriber', 'Site');

		// Check out the item
		if ($editId)
		{
			$model->checkout($editId);
		}

		// Check in the previous user.
		if ($previousId && $previousId !== $editId)
		{
			$model->checkin($previousId);
		}

		// Redirect to the edit screen.
		$this->setRedirect(Route::_('index.php?option=com_subscription&view=subscriberform&layout=edit', false));
	}
	
	public function cancel(){
		
		$user = $this->app->getIdentity();
		$editId     = $this->input->getInt('subscription_id', 0);
		$db = Factory::getContainer()->get('DatabaseDriver');
		$db->setQuery("SELECT * FROM #__subscription_subscribers WHERE user_id=".$user->get('id')." AND id=".$editId);
		$item = $db->loadObject();
		
		if ($item->id>0){
			
			
			$obj = SubscriptionHelper::getStripeKeys($item->product_id);
			
			$stripe = new \Stripe\StripeClient($obj->server);
			
			//lets remove the subscription
			
			$subscription = $stripe->subscriptions->retrieve($item->stripe_subscription_id);
			$subscription->delete();
			
			$item->state=0;
			$item->end_date = date("Y-m-d");
			
			$db->updateObject("#__subscription_subscribers",$item,"id");
			
		}
		
		
		
	    $this->setRedirect(Route::_('index.php?option=com_subscription&view=subscriptions', false));
	}

	/**
	 * Method to save data
	 *
	 * @return    void
	 *
	 * @throws  Exception
	 * @since   1.0.0
	 */
	public function publish()
	{
		// Checking if the user can remove object
		$user = $this->app->getIdentity();

		if ($user->authorise('core.edit', 'com_subscription') || $user->authorise('core.edit.state', 'com_subscription'))
		{
			$model = $this->getModel('Subscriber', 'Site');

			// Get the user data.
			$id    = $this->input->getInt('id');
			$state = $this->input->getInt('state');

			// Attempt to save the data.
			$return = $model->publish($id, $state);

			// Check for errors.
			if ($return === false)
			{
				$this->setMessage(Text::sprintf('Save failed: %s', $model->getError()), 'warning');
			}

			// Clear the profile id from the session.
			$this->app->setUserState('com_subscription.edit.subscriber.id', null);

			// Flush the data from the session.
			$this->app->setUserState('com_subscription.edit.subscriber.data', null);

			// Redirect to the list screen.
			$this->setMessage(Text::_('COM_SUBSCRIPTION_ITEM_SAVED_SUCCESSFULLY'));
			$menu = Factory::getApplication()->getMenu();
			$item = $menu->getActive();

			if (!$item)
			{
				// If there isn't any menu item active, redirect to list view
				$this->setRedirect(Route::_('index.php?option=com_subscription&view=subscribers', false));
			}
			else
			{
				$this->setRedirect(Route::_('index.php?Itemid='. $item->id, false));
			}
		}
		else
		{
			throw new \Exception(500);
		}
	}

	/**
	 * Check in record
	 *
	 * @return  boolean  True on success
	 *
	 * @since   1.0.0
	 */
	public function checkin()
	{
		// Check for request forgeries.
		$this->checkToken('GET');

		$id 	= $this->input->getInt('id', 0);
		$model 	= $this->getModel();
		$item 	= $model->getItem($id);

		// Checking if the user can remove object
		$user = $this->app->getIdentity();

		if ($user->authorise('core.manage', 'com_subscription') || $item->checked_out == $user->id) { 

			$return = $model->checkin($id);

			if ($return === false)
			{
				// Checkin failed.
				$message = Text::sprintf('JLIB_APPLICATION_ERROR_CHECKIN_FAILED', $model->getError());
				$this->setRedirect(Route::_('index.php?option=com_subscription&view=subscriber' . '&id=' . $id, false), $message, 'error');
				return false;
			}
			else
			{
				// Checkin succeeded.
				$message = Text::_('COM_SUBSCRIPTION_CHECKEDIN_SUCCESSFULLY');
				$this->setRedirect(Route::_('index.php?option=com_subscription&view=subscriber' . '&id=' . $id, false), $message);
				return true;
			}
		}
		else
		{
			throw new \Exception(Text::_('JERROR_ALERTNOAUTHOR'), 403);
		}
	}

	/**
	 * Remove data
	 *
	 * @return void
	 *
	 * @throws Exception
	 */
	public function remove()
	{
		// Checking if the user can remove object
		$user = $this->app->getIdentity();

		if ($user->authorise('core.delete', 'com_subscription'))
		{
			$model = $this->getModel('Subscriber', 'Site');

			// Get the user data.
			$id = $this->input->getInt('id', 0);

			// Attempt to save the data.
			$return = $model->delete($id);

			// Check for errors.
			if ($return === false)
			{
				$this->setMessage(Text::sprintf('Delete failed', $model->getError()), 'warning');
			}
			else
			{
				// Check in the profile.
				if ($return)
				{
					$model->checkin($return);
				}

				$this->app->setUserState('com_subscription.edit.subscriber.id', null);
				$this->app->setUserState('com_subscription.edit.subscriber.data', null);

				$this->app->enqueueMessage(Text::_('COM_SUBSCRIPTION_ITEM_DELETED_SUCCESSFULLY'), 'success');
				$this->app->redirect(Route::_('index.php?option=com_subscription&view=subscribers', false));
			}

			// Redirect to the list screen.
			$menu = Factory::getApplication()->getMenu();
			$item = $menu->getActive();
			$this->setRedirect(Route::_($item->link, false));
		}
		else
		{
			throw new \Exception(500);
		}
	}
}
