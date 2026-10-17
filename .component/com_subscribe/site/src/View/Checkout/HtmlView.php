<?php

/**
 * @version    CVS: 1.0.0
 * @package    Com_Subscription
 * @author     Rob <pr@floridarangers.com>
 * @copyright  2024 Rob
 * @license    GNU General Public License version 2 or later; see LICENSE.txt
 */

namespace Sub\Component\Subscription\Site\View\Checkout;
// No direct access
defined('_JEXEC') or die;

use Joomla\CMS\MVC\View\HtmlView as BaseHtmlView;
use \Joomla\CMS\Factory;
use \Joomla\CMS\Language\Text;
use Sub\Component\Subscription\Site\Helper\SubscriptionHelper;
require_once(JPATH_ADMINISTRATOR."/components/com_payage/helpers/Stripe/init.php");

/**
 * View class for a list of Subscription.
 *
 * @since  1.0.0
 */
class HtmlView extends BaseHtmlView
{
	protected $state;

	protected $item;

	protected $form;

	protected $params;

	/**
	 * Display the view
	 *
	 * @param   string  $tpl  Template name
	 *
	 * @return void
	 *
	 * @throws Exception
	 */
	public function display($tpl = null)
	{
		$app  = Factory::getApplication();
		$user = $app->getIdentity();

		$this->state  = $this->get('State');
		$this->item   = $this->get('Item');
		$this->params = $app->getParams('com_subscription');

		if (empty($this->item))
		{
			throw new \Exception("Missing Checkout Data");
		}

		// Check for errors.
		if (count($errors = $this->get('Errors')))
		{
			throw new \Exception(implode("\n", $errors));
		}
		//check if we have a subscription arleady
		
		$db = Factory::getContainer()->get('DatabaseDriver');
		
		$db->setQuery("SELECT * FROM #__subscription_subscribers WHERE state=1 AND product_id=".$this->item->id." AND user_id=".$user->get('id'));
		$sub = $db->loadObject();
		
		//we already have an active sub!
		if($sub->id>0){
		
		    $app->enqueueMessage("You already have an active subscription");
		   	$app->redirect('index.php?option=com_subscription&view=subscriber&id='.$sub->id);
			return;
			
			
		}
		
	
	    $obj = SubscriptionHelper::getStripeKeysMode($this->item->test_mode);

		if ($user->get('id')<1) {
			
			throw new \Exception("You must be logged in to continue");
			
		}



		
		$this->data = new \stdClass;

		$stripe = new \Stripe\StripeClient($obj->server);
		
		$customer = SubscriptionHelper::getStripeCustomer($this->item->id);
				  
				 
		
		// Create the subscription with the customer ID, price ID, and necessary options.
		$subscription = $stripe->subscriptions->create([
			'customer' => $customer->id,
			'items' => [[
				'price' => $this->item->stripe_product_id,
			]],
			'payment_behavior' => 'default_incomplete',
			'payment_settings' => ['save_default_payment_method' => 'on_subscription'],
			'expand' => ['latest_invoice.payment_intent'],
		]);
		
		  
		  $this->data->client_secret =  $subscription->latest_invoice->payment_intent->client_secret;
		  $this->data->product_id =  $this->item->stripe_product_id;
		  $this->data->customer_id = $customer->id;
		  $this->data->subscription_id = $subscription->id;
		  
		  	  
		  $wa = Factory::getApplication()->getDocument()->getWebAssetManager();
		  $wa->useScript('keepalive');
		  Factory::getApplication()->getDocument()->addScript("https://js.stripe.com/v3/");
		  
		  $wa->addInlineScript("
		  
		  document.addEventListener('DOMContentLoaded', async () => {
			 stripe = Stripe('".($obj->js)."');
		  });
		   
		  ");
		  

		if ($this->_layout == 'edit')
		{
			$authorised = $user->authorise('core.create', 'com_subscription');

			if ($authorised !== true)
			{
				throw new \Exception(Text::_('JERROR_ALERTNOAUTHOR'));
			}
		}

		$this->_prepareDocument();

		parent::display($tpl);
	}

	/**
	 * Prepares the document
	 *
	 * @return void
	 *
	 * @throws Exception
	 */
	protected function _prepareDocument()
	{
		$app   = Factory::getApplication();
		$menus = $app->getMenu();
		$title = null;

		// Because the application sets a default page title,
		// We need to get it from the menu item itself
		$menu = $menus->getActive();

		if ($menu)
		{
			$this->params->def('page_heading', $this->params->get('page_title', $menu->title));
		}
		else
		{
			$this->params->def('page_heading', Text::_('COM_SUBSCRIPTION_DEFAULT_PAGE_TITLE'));
		}

		$title = $this->params->get('page_title', '');

		if (empty($title))
		{
			$title = $app->get('sitename');
		}
		elseif ($app->get('sitename_pagetitles', 0) == 1)
		{
			$title = Text::sprintf('JPAGETITLE', $app->get('sitename'), $title);
		}
		elseif ($app->get('sitename_pagetitles', 0) == 2)
		{
			$title = Text::sprintf('JPAGETITLE', $title, $app->get('sitename'));
		}

		$this->document->setTitle($title);

		if ($this->params->get('menu-meta_description'))
		{
			$this->document->setDescription($this->params->get('menu-meta_description'));
		}

		if ($this->params->get('menu-meta_keywords'))
		{
			$this->document->setMetadata('keywords', $this->params->get('menu-meta_keywords'));
		}

		if ($this->params->get('robots'))
		{
			$this->document->setMetadata('robots', $this->params->get('robots'));
		}

		
			// Add Breadcrumbs
			$pathway = $app->getPathway();
						$breadcrumbTitle = Text::_('COM_SUBSCRIPTION_TITLE_CHECKOUT');
			
						if(!in_array($breadcrumbTitle, $pathway->getPathwayNames())) {
							$pathway->addItem($breadcrumbTitle);    
						}
				
	}
}
